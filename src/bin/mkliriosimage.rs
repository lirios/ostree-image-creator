/****************************************************************************
 * Copyright (C) 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 ***************************************************************************/

extern crate mkliriosimage;

use clap::{App, Arg};
use env_logger;
use libc;
use log::error;
use std::env;
use std::path::Path;
use std::process;
use uname;

use mkliriosimage::builder;

fn main() {
    if env::var("RUST_LOG").is_err() {
        env::set_var("RUST_LOG", "info");
    }
    env_logger::builder()
        .format_timestamp(None)
        .format_level(false)
        .format_module_path(false)
        .init();

    unsafe {
        if libc::geteuid() != 0 {
            error!("Please run this as root!");
            process::exit(1);
        }
    }

    // Determine architecture
    let arch = match uname::uname() {
        Err(why) => {
            error!("{}", why);
            process::exit(1);
        }
        Ok(utsinfo) => {
            utsinfo.machine.to_owned()
        }
    };

    // Command line arguments
    let matches = App::new("mkliriosimage")
        .version("1.0")
        .about("Creates images of Liri OS.")
        .author("Pier Luigi Fiorini")
        .arg(
            Arg::with_name("workdir")
                .short("w")
                .long("workdir")
                .value_name("DIRECTORY")
                .default_value("/var/tmp")
                .required(true)
                .takes_value(true)
                .help("Path where all temporary files are created"),
        )
        .arg(
            Arg::with_name("configdir")
                .short("c")
                .long("configdir")
                .value_name("DIRECTORY")
                .takes_value(true)
                .help(
                    "Directory that contains media configuration files to be copied into the image",
                ),
        )
        .arg(
            Arg::with_name("manifest")
                .short("m")
                .long("manifest")
                .value_name("FILE")
                .required(true)
                .takes_value(true)
                .help("A YAML file with the image definition"),
        )
        .arg(
            Arg::with_name("fslabel")
                .long("fslabel")
                .value_name("LABEL")
                .takes_value(true)
                .help("File system label for live images (max 32 bytes)"),
        )
        .arg(
            Arg::with_name("force")
                .short("f")
                .long("force")
                .help("Overwrite previously generated files"),
        )
        .arg(
            Arg::with_name("output")
                .short("o")
                .long("output")
                .takes_value(true)
                .help("Name of the image file to create"),
        )
        .get_matches();

    if matches.is_present("configdir") {
        let configdir = matches.value_of("configdir").unwrap();
        let config_path = Path::new(configdir);
        if !config_path.exists() {
            error!("Media configuration path does not exist");
            process::exit(1);
        }
        if !config_path.is_dir() {
            error!("Media configuration path is not a directory");
            process::exit(1);
        }
        if !config_path.join("README-devel.md").exists() {
            error!("Invalid media configuration directory");
            process::exit(1);
        }
    }

    match builder::Manifest::from_file(Path::new(matches.value_of("manifest").unwrap())) {
        Err(why) => {
            error!("{}", why);
            process::exit(1);
        }
        Ok(manifest) => {
            if manifest.image_type != builder::ImageType::Live && matches.is_present("fslabel") {
                error!("Use --fslabel only for live images");
                process::exit(1);
            }

            let configdir = if matches.is_present("configdir") {
                Some(matches.value_of("configdir").unwrap())
            } else {
                None
            };

            mkliriosimage::start(
                &manifest,
                &arch,
                matches.value_of("workdir").unwrap(),
                configdir,
                matches.value_of("output"),
                matches.is_present("force"),
            );
        }
    }
}
