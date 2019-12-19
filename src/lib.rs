/****************************************************************************
 * Copyright (C) 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 ***************************************************************************/

extern crate bytefmt;
extern crate chrono;
extern crate clap;
extern crate env_logger;
#[macro_use]
extern crate failure;
extern crate nix;
extern crate log;
extern crate serde;
extern crate shell_words;
extern crate tempfile;
extern crate tera;
extern crate uname;

use crate::builder::Builder;
use crate::builder::Manifest;

use log::error;
use std::process;

pub mod builder;
pub mod cmd;
pub mod diskimage;
mod ostree;
pub mod templates;

pub fn start(
    manifest: &Manifest,
    arch: &str,
    workdir: &str,
    configdir: Option<&str>,
    filename: Option<&str>,
    force: bool,
) {
    match manifest.image_type {
        builder::ImageType::Disk => {
            let _builder =
                builder::DiskBuilder::new(&arch, &workdir, configdir, filename, force, &manifest);
            match _builder.build() {
                Err(why) => {
                    error!("{}", why);
                    process::exit(1);
                }
                Ok(()) => {
                    process::exit(0);
                }
            }
        }
        builder::ImageType::Live => {
            let _builder = builder::LiveBuilder::new(
                &arch, &workdir, configdir, filename, None, force, &manifest,
            );
            match _builder.build() {
                Err(why) => {
                    error!("{}", why);
                    process::exit(1);
                }
                Ok(()) => {
                    process::exit(0);
                }
            }
        }
    }
}
