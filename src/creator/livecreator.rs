/****************************************************************************
 * Copyright (C) 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 ***************************************************************************/

#[allow(unused_macros)]
macro_rules! step {
    ($fmt:expr) => (print!(concat!("‣ \x1b[0;1;39m", $fmt, "\x1b[0m\n")));
    ($fmt:expr, $($arg:tt)*) => (print!(concat!("‣ \x1b[0;1;39m", $fmt, "\x1b[0m\n"), $($arg)*));
}

use crate::cmd;
use crate::creator::BuildError;
use crate::creator::BuildResult;
use crate::creator::Creator;
use crate::creator::Manifest;
use crate::diskimage::DiskImage;
use crate::ostree;
use crate::templates;

use bytefmt;
use log::{debug, error, info};
use nix::unistd;
use std::convert::From;
use std::env;
use std::fs;
use std::io::{self, BufReader, Read, Write};
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::process;
use tempfile::TempDir;

// Convention for kernel and initramfs names
static KERNEL_IMG: &'static str = "vmlinuz";
static INITRAMFS_IMG: &'static str = "initramfs.img";

pub struct LiveCreator {
    workdir: String,
    configdir: Option<String>,
    filename: String,
    volset: String,
    fslabel: String,
    live_title: String,
    live_product: String,
    live_timeout: u32,
    squashfs_compression: String,
    extra_kargs: Vec<String>,
    force: bool,
    osname: String,
    arch: String,
    size: u64,
    selinux: bool,
    repodir: PathBuf,
    repo_is_local: bool,
    remote_url: String,
    refspec: String,
    refs: Vec<String>,
}

impl LiveCreator {
    pub fn new(
        arch: &str,
        workdir: &str,
        configdir: Option<&str>,
        filename: Option<&str>,
        fslabel: Option<&str>,
        force: bool,
        manifest: &Manifest,
    ) -> LiveCreator {
        // Parse size from manifest
        let size = match bytefmt::parse(&manifest.size) {
            Err(why) => {
                error!("Cannot parse \"size\" value from manifest: {}", why);
                process::exit(1);
            }
            Ok(result) => result,
        };

        // Determine volume label
        let now = chrono::Utc::now();
        let volset = format!(
            "{}-{}-{}",
            &manifest.osname,
            now.format("%Y%m%d%H%M"),
            &arch
        );
        let default_iso_filename = format!("{}.iso", &volset);
        let iso_filename = filename.unwrap_or(&default_iso_filename);
        let mut iso_fslabel = fslabel.unwrap_or(&volset);
        if iso_fslabel.len() > 32 {
            iso_fslabel = iso_fslabel.split_at(32).0;
        }

        // We directly use local remotes, otherwise we create a repository and
        // add a remote
        let is_local =
            manifest.remote_url.starts_with("/") || manifest.remote_url.starts_with("file://");
        let repodir = if is_local {
            PathBuf::from(&manifest.remote_url.replace("file://", ""))
        } else {
            PathBuf::from(&workdir).join("repo")
        };

        LiveCreator {
            workdir: workdir.to_string(),
            configdir: configdir.map(|s| s.to_string()),
            filename: iso_filename.to_string(),
            volset: volset.to_owned(),
            fslabel: iso_fslabel.to_string(),
            live_title: manifest.live.title.to_owned(),
            live_product: manifest.live.product.to_owned(),
            live_timeout: manifest.live.timeout,
            squashfs_compression: manifest.live.squashfs_compression.to_owned(),
            extra_kargs: manifest.extra_kargs.to_owned(),
            force: force,
            osname: manifest.osname.to_owned(),
            arch: arch.to_string(),
            size: size,
            selinux: manifest.selinux,
            repodir: repodir,
            repo_is_local: is_local,
            remote_url: manifest.remote_url.to_owned(),
            refspec: manifest.main_ref.replace("${basearch}", &arch),
            refs: manifest
                .refs
                .iter()
                .map(|s| s.replace("${basearch}", &arch))
                .collect(),
        }
    }

    fn visit_dirs<F>(&self, dir: &Path, func: &mut F) -> io::Result<()>
    where
        F: FnMut(&fs::DirEntry),
    {
        if dir.is_dir() {
            for entry in fs::read_dir(dir)? {
                let entry = entry?;
                let path = entry.path();
                if path.is_dir() {
                    self.visit_dirs(&path, func)?;
                } else {
                    func(&entry);
                }
            }
        }
        Ok(())
    }

    fn estimate_directory_size(
        &self,
        start_path: &Path,
        add_percent: Option<u64>,
    ) -> io::Result<u64> {
        let mut total_size: u64 = 0;
        self.visit_dirs(&start_path, &mut |entry: &fs::DirEntry| {
            if let Ok(metadata) = entry.metadata() {
                total_size += metadata.len();
            }
        })?;
        let add_percent_modifier = (100.0 + add_percent.unwrap_or(5) as f32) / 100.0;
        total_size = 1 + (total_size as f32 * add_percent_modifier) as u64;
        Ok(total_size)
    }

    fn create_rootfs(&self, tmp_path: &Path) -> BuildResult {
        step!("Creating root file system");

        let dir_path = tmp_path.join("squashfs").join("LiveOS");
        let file_path = dir_path.join("rootfs.img");
        let mountpoint_path = tmp_path.join("root");

        fs::create_dir_all(&dir_path)?;

        // Mount disk image
        let disk = DiskImage::new(&file_path, &mountpoint_path);
        disk.create(self.size)?;
        disk.format("ext4")?;
        disk.mount()?;

        // Pull and deploy OS tree
        let ostreedir = mountpoint_path.join("ostree");
        let repodir = ostreedir.join("repo");
        let deploydir = ostreedir.join("deploy");
        fs::create_dir_all(&repodir)?;
        fs::create_dir_all(&deploydir)?;
        step!("Pulling OS tree(s)");
        ostree::init(&repodir, ostree::OstreeArchiveMode::Bare)?;
        ostree::pull_local(&self.repodir, &repodir, &self.refs)?;
        step!("Deploying OS tree");
        ostree::os_init(&self.osname, &mountpoint_path)?;
        ostree::deploy(&self.osname, &self.refspec, &mountpoint_path)?;

        // Create a few directories under /var and label /var/home to make SELinux happy
        // https://github.com/coreos/ignition-dracut/pull/79#issuecomment-488446949
        let vardir = deploydir.join(&self.osname).join("var");
        for dirname in &["home", "log/journal", "lib/systemd"] {
            fs::create_dir_all(vardir.join(&dirname))?;
        }
        let homedir = vardir.join("home");
        if self.selinux {
            let label = cmd::check_output(&["matchpathcon", "-n", "/home"])?.to_string();
            cmd::run(&[
                "chcon",
                &label,
                &homedir.into_os_string().into_string().unwrap(),
            ])?;
        }

        Ok(())
    }

    fn create_squashfs(&self, liveos_path: &Path, root_path: &Path) -> BuildResult {
        step!("Compressing squashfs with {}", &self.squashfs_compression);

        fs::create_dir_all(&liveos_path)?;

        let squashfs_path = liveos_path.join("squashfs.img");
        cmd::run_with_cwd(
            &[
                "mksquashfs",
                ".",
                &squashfs_path.into_os_string().into_string().unwrap(),
                "-comp",
                &self.squashfs_compression,
            ],
            &root_path,
        )?;

        Ok(())
    }

    fn create_efiboot(&self, dir_path: &Path, filename: &Path, mountpoint: &Path) -> BuildResult {
        step!("Creating EFI boot image");

        // Estimate directory size
        let size = self.estimate_directory_size(&dir_path, Some(25))?;

        // Mount the image file (will be unmounted when it goes out of scope)
        let disk = DiskImage::new(&filename, &mountpoint);
        disk.create(size)?;
        disk.format("msdos")?;
        disk.mount()?;

        // Copy files
        let destdir = mountpoint.join("EFI");
        fs::create_dir_all(&destdir)?;
        cmd::run_with_cwd(
            &[
                "cp",
                "-R",
                "-L",
                "--preserve=timestamps",
                ".",
                &destdir.into_os_string().into_string().unwrap(),
            ],
            &dir_path,
        )?;

        Ok(())
    }

    fn copy_syslinux(&self, tmp_isolinux: &Path) -> BuildResult {
        step!("Copying syslinux files to ISO");
        for filename in &[
            "/usr/share/syslinux/isolinux.bin",
            "/usr/share/syslinux/ldlinux.c32",
            "/usr/share/syslinux/libcom32.c32",
            "/usr/share/syslinux/libutil.c32",
            "/usr/share/syslinux/vesamenu.c32",
        ] {
            let src_path = Path::new(&filename);
            let dst_path = &tmp_isolinux.join(src_path.file_name().unwrap());
            debug!("Copy {:?} -> {:?}", &src_path, &dst_path);
            fs::copy(&src_path, &dst_path)?;

            let mut perms = fs::metadata(&dst_path)?.permissions();
            perms.set_mode(0o755);
        }

        Ok(())
    }
}

impl Creator for LiveCreator {
    fn build(&self) -> BuildResult {
        if Path::new(&self.filename).exists() && !self.force {
            error!(
                "File {} already exist, you can force a rebuild passing `--force`",
                self.filename
            );
            process::exit(1);
        }

        info!("           Work directory: {}", self.workdir);
        info!("  Configuration directory: {:?}", self.configdir);
        info!("                   Volume: {}", self.volset);
        info!("        File system label: {}", self.fslabel);
        info!("         Output file name: {}", self.filename);
        info!("                    Force: {}", self.force);
        info!("             Architecture: {}", self.arch);

        // Create temporary directory for the build
        step!("Setting up temporary workspace");
        fs::create_dir_all(&self.workdir)?;
        let tmp_dir = TempDir::new_in(Path::new(&self.workdir))?;
        step!("Created temporary workspace in {:?}", &tmp_dir.path());

        // Create empty directories for the process
        let tmp_isoroot = tmp_dir.path().join("iso");
        let tmp_isoimages = tmp_isoroot.join("images");
        let tmp_isolinux = tmp_isoroot.join("isolinux");
        let tmp_efidir = tmp_isoroot.join("EFI").join("fedora");
        fs::create_dir_all(&tmp_isoimages)?;
        fs::create_dir_all(&tmp_isolinux)?;
        fs::create_dir_all(&tmp_efidir)?;

        // Temporary ISO file
        let tmp_isofile = tmp_dir
            .path()
            .join("output.iso")
            .into_os_string()
            .into_string()
            .unwrap()
            .to_owned();

        // OS tree
        if !self.repo_is_local {
            step!("Mirroring OSTree repository");
            ostree::init(&self.repodir, ostree::OstreeArchiveMode::Archive)?;
            ostree::remote_add(&self.repodir, &self.osname, &self.remote_url)?;
            for refspec in &self.refs {
                ostree::mirror(&self.repodir, &self.osname, &refspec)?;
            }
        }

        // Resolve refspec
        let commit = ostree::resolve_rev(&self.repodir, &self.refspec)?;
        step!("Resolved {} to {}", &self.refspec, &commit);

        // Find the directory under `/usr/lib/modules/<kver>` where the
        // kernel/initrd live. It will be the 2nd entity output by
        // `ostree ls <commit> /usr/lib/modules`
        let _path = ostree::list(&self.repodir, Path::new("/usr/lib/modules"), &commit)?.nth(1);
        if _path.is_none() {
            return Err(BuildError::Error(String::from(
                "Kernel modules directory not found",
            )));
        }
        let moduledir = _path.unwrap().to_owned();

        // Copy those files from the OS tree to the ISO root dir
        step!("Extracting kernel and initramfs");
        for filename in &[KERNEL_IMG, INITRAMFS_IMG] {
            ostree::checkout(
                &self.repodir,
                Path::new(&moduledir).join(filename).as_path(),
                tmp_isoimages.as_path(),
                &commit,
            )?;

            // initramfs isn't world readable by default so let's open up perms
            let mut perms = fs::metadata(tmp_isoimages.join(filename))?.permissions();
            perms.set_mode(0o755);
        }

        // Copy memtest from `/usr/lib/ostree-boot` using a glob because there's an always
        // changing version in the file name
        let mut has_memtest = false;
        step!("Extracting memtest");
        for filename in ostree::list(&self.repodir, Path::new("/usr/lib/ostree-boot"), &commit)? {
            let path = Path::new(&filename);
            if let Some(basename) = path.file_name() {
                let basename_str = basename.to_str().unwrap_or_default();
                if basename_str.starts_with("memtest86+") {
                    has_memtest = true;
                    ostree::checkout(
                        &self.repodir,
                        Path::new(&filename),
                        tmp_isoimages.as_path(),
                        &commit,
                    )?;
                    let src_path = tmp_isoimages.join(&basename_str);
                    let dst_path = tmp_isoimages.join("memtest");
                    debug!("Move {:?} -> {:?}", &src_path, &dst_path);
                    fs::rename(&src_path, &dst_path)?;
                    break;
                }
            }
        }

        // See if checkisomd5 is available, so we add an entry to the bootloader
        let has_checkisomd5 =
            ostree::list(&self.repodir, Path::new("/usr/bin/checkisomd5"), &commit)?.len() > 0;

        // Create rootfs
        self.create_rootfs(&tmp_dir.path())?;

        // Compress squashfs
        self.create_squashfs(
            &tmp_isoroot.join("LiveOS"),
            &tmp_dir.path().join("squashfs"),
        )?;

        // Add extra kernel arguments
        let mut kargs_list: Vec<String> = self.extra_kargs.to_owned();
        for karg in &["quiet", "rd.live.image"] {
            if !kargs_list.contains(&karg.to_string()) {
                kargs_list.push(karg.to_string());
            }
        }
        let kargs = kargs_list.join(" ");

        // Grab all the contents from the image configuration project
        if self.configdir.is_some() {
            step!("Copying files to ISO");
            let config_path = Path::new(self.configdir.as_ref().unwrap());
            for entry in fs::read_dir(&config_path)? {
                let entry = entry?;
                let path = entry.path();
                let dir_suffix = path.strip_prefix(&config_path);
                if dir_suffix.is_ok() {
                    let basename = dir_suffix.unwrap();
                    let dst_path = tmp_isoroot.join(&basename);

                    if path.is_dir() {
                        fs::create_dir_all(&dst_path)?;
                    } else {
                        // Skip development files
                        if basename.starts_with("README-devel.md") {
                            continue;
                        }

                        // Create file (assume all files are text)
                        let src_file = fs::File::open(&path)?;
                        let mut buf_reader = BufReader::new(src_file);
                        let mut contents = String::new();
                        buf_reader.read_to_string(&mut contents)?;
                        contents = contents
                            .replace("@@FSLABEL@@", &self.fslabel)
                            .replace("@@KERNEL-ARGS@@", &kargs);
                        let mut dst_file = fs::File::create(&dst_path)?;
                        println!("{:?} -> {:?}", &path, &dst_path);
                        dst_file.write_all(contents.as_bytes())?;
                    }
                }
            }
        }

        // Generate boot loader configuration
        let mut t = templates::Template::new(
            &self.live_product,
            &self.live_title,
            &self.fslabel,
            self.live_timeout,
        )?;
        t.add_stanza("linux", &format!("^Start {}", &self.live_product), &kargs);
        if has_checkisomd5 {
            t.add_stanza(
                "check",
                &format!("Test this ^media & start {}", &self.live_product),
                &format!("{} rd.live.check", &kargs),
            );
        }
        t.set_vesa_kargs(&kargs);
        t.enable_memtest(has_memtest);
        let grub_text = t.render(templates::TemplateType::Grub)?;
        fs::write(&tmp_efidir.join("grub.cfg"), grub_text)?;
        let syslinux_text = t.render(templates::TemplateType::Syslinux)?;
        fs::write(&tmp_isolinux.join("isolinux.cfg"), syslinux_text)?;

        // Generate the ISO image. Lots of good info here:
        // https://fedoraproject.org/wiki/User:Pjones/BootableCDsForBIOSAndUEFI
        let mut genisoargs = vec![
            "genisoimage",
            "-verbose",
            "-V",
            &*self.fslabel,
            "-volset",
            &*self.volset,
            "-rational-rock",
            "-J",
            "-joliet-long",
        ];

        // For x86_64 legacy boot (BIOS) booting
        if self.arch == String::from("x86_64") {
            // Install binaries from syslinux
            self.copy_syslinux(&tmp_isolinux)?;

            // For legacy bios boot AKA eltorito boot
            genisoargs.push("-eltorito-boot");
            genisoargs.push("isolinux/isolinux.bin");
            genisoargs.push("-eltorito-catalog");
            genisoargs.push("isolinux/boot.cat");
            genisoargs.push("-no-emul-boot");
            genisoargs.push("-boot-load-size");
            genisoargs.push("4");
            genisoargs.push("-boot-info-table");
        }

        // For x86_64 and aarch64 UEFI booting
        match self.arch.as_ref() {
            "x86_64" | "aarch64" => {
                // Create the efiboot.img file. This is a fat32 formatted
                // filesystem that contains all the files needed for EFI boot
                // from an ISO.
                step!("Extracting EFI files");
                let imageefidir = tmp_dir.path().join("efi");
                let efibootfile = tmp_isoimages.join("efiboot.img");
                ostree::checkout(
                    &self.repodir,
                    Path::new("/usr/lib/ostree-boot/efi/EFI"),
                    &imageefidir,
                    &commit,
                )?;
                self.create_efiboot(&imageefidir, &efibootfile, &tmp_dir.path().join("efiboot"))?;
                genisoargs.push("-eltorito-alt-boot");
                genisoargs.push("-efi-boot");
                genisoargs.push("images/efiboot.img");
                genisoargs.push("-no-emul-boot");
            }
            _ => {}
        };

        // Define inputs and outputs
        genisoargs.push("-o");
        genisoargs.push(&tmp_isofile);
        genisoargs.push(".");

        // Create ISO
        step!("Creating ISO image");
        cmd::run_with_cwd(&genisoargs, &tmp_isoroot)?;

        // Add MBR for x86_64 legacy (BIOS) boot when ISO is copied to a USB stick
        if self.arch == String::from("x86_64") {
            step!("Running isohybrid");
            cmd::run(&["isohybrid", &tmp_isofile])?;
        }

        // Implant MD5 for checksum check
        if has_checkisomd5 {
            step!("Implanting MD5 checksum in ISO image");
            cmd::run(&["implantisomd5", &tmp_isofile])?;
        }

        // Change ownership if we are running with sudo
        let sudo_uid = env::var("SUDO_UID")
            .unwrap_or_default()
            .parse::<u32>()
            .unwrap_or(0);
        let sudo_gid = env::var("SUDO_GID")
            .unwrap_or_default()
            .parse::<u32>()
            .unwrap_or(0);
        if sudo_uid > 0 && sudo_gid > 0 {
            step!("Changing file ownership to {}:{}", &sudo_uid, &sudo_gid);
            match unistd::chown(
                Path::new(&tmp_isofile),
                Some(unistd::Uid::from_raw(sudo_uid)),
                Some(unistd::Gid::from_raw(sudo_gid)),
            ) {
                Err(why) => error!("Failed to change ownership: {:?}", why),
                Ok(()) => {}
            }
        }

        // Move the file where the user expects it to be
        fs::rename(&tmp_isofile, &self.filename)?;
        println!("Wrote: {}", &self.filename);

        Ok(())
    }
}
