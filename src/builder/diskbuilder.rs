/****************************************************************************
 * Copyright (C) 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 ***************************************************************************/

use crate::builder::BuildResult;
use crate::builder::Builder;
use crate::builder::Manifest;

use log::{error, info};
use std::path::Path;
use std::process;

pub struct DiskBuilder {
    workdir: String,
    configdir: Option<String>,
    filename: String,
    force: bool,
    arch: String,
}

impl DiskBuilder {
    pub fn new(
        arch: &str,
        workdir: &str,
        configdir: Option<&str>,
        filename: Option<&str>,
        force: bool,
        manifest: &Manifest,
    ) -> DiskBuilder {
        let now = chrono::Utc::now();

        DiskBuilder {
            workdir: workdir.to_string(),
            configdir: configdir.map(|s| s.to_string()),
            filename: filename
                .unwrap_or(&format!(
                    "{}-{}-{}",
                    &manifest.osname,
                    now.format("%Y%m%d%H%M"),
                    &arch
                ))
                .to_string(),
            force: force,
            arch: arch.to_string(),
        }
    }
}

impl Builder for DiskBuilder {
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
        info!("         Output file name: {}", self.filename);
        info!("                    Force: {}", self.force);
        info!("             Architecture: {}", self.arch);

        Ok(())
    }
}
