/****************************************************************************
 * Copyright (C) 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 ***************************************************************************/

use crate::cmd;

use std::fs;
use std::io::Result;
use std::path::Path;

pub struct DiskImage {
    filename: String,
    mountpoint: String,
}

impl DiskImage {
    pub fn new(file_path: &Path, mountpoint_path: &Path) -> DiskImage {
        DiskImage {
            filename: file_path.to_str().unwrap_or_default().to_string(),
            mountpoint: mountpoint_path.to_str().unwrap_or_default().to_string(),
        }
    }

    pub fn create(&self, size: u64) -> Result<()> {
        let file = fs::File::create(&self.filename)?;
        file.set_len(size)
    }

    pub fn format(&self, fs: &str) -> cmd::CommandResult<()> {
        cmd::run(&[&format!("mkfs.{}", &fs), &self.filename])
    }

    pub fn mount(&self) -> cmd::CommandResult<()> {
        fs::create_dir_all(&self.mountpoint)?;
        cmd::run(&["mount", "-n", &self.filename, &self.mountpoint, "-oloop"])
    }

    pub fn umount(&self) -> cmd::CommandResult<()> {
        cmd::no_output(&["umount", "--recursive", "-n", &self.mountpoint])
    }
}

impl Drop for DiskImage {
    #[allow(unused_must_use)]
    fn drop(&mut self) {
        // Unmount the file system and ignore errors
        self.umount();
    }
}
