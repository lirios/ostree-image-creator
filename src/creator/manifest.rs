/****************************************************************************
 * Copyright (C) 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 ***************************************************************************/

use serde::Deserialize;
use serde_yaml;
use std::fs;
use std::path;

#[derive(Deserialize, Debug, PartialEq)]
pub enum ImageType {
    #[serde(rename = "disk")]
    Disk,

    #[serde(rename = "live")]
    Live,
}

#[derive(Deserialize, Default, Debug)]
pub struct LiveOptions {
    #[serde(default = "default_live_title")]
    pub title: String,

    #[serde(default = "default_live_product")]
    pub product: String,

    #[serde(default = "default_live_timeout")]
    pub timeout: u32,

    #[serde(rename(deserialize = "squashfs-compression"))]
    #[serde(default = "default_live_squashfs_compression")]
    pub squashfs_compression: String,
}

#[derive(Deserialize, Debug)]
pub struct Manifest {
    #[serde(rename(deserialize = "type"))]
    pub image_type: ImageType,

    #[serde(default = "default_size")]
    pub size: String,

    #[serde(default = "default_selinux")]
    pub selinux: bool,

    pub osname: String,

    #[serde(rename(deserialize = "main-ref"))]
    pub main_ref: String,

    #[serde(default)]
    pub refs: Vec<String>,

    #[serde(rename(deserialize = "remote-url"))]
    pub remote_url: String,

    #[serde(rename(deserialize = "extra-kargs"))]
    #[serde(default)]
    pub extra_kargs: Vec<String>,

    #[serde(default)]
    pub live: LiveOptions,
}

fn default_selinux() -> bool {
    false
}

fn default_size() -> String {
    "4 GiB".to_string()
}

fn default_live_title() -> String {
    "Linux".to_string()
}

fn default_live_product() -> String {
    "Linux (Live)".to_string()
}

fn default_live_timeout() -> u32 {
    600
}

fn default_live_squashfs_compression() -> String {
    "zstd".to_string()
}

impl Manifest {
    pub fn from_file(path: &path::Path, repo: Option<&str>) -> Result<Manifest, Box<dyn std::error::Error + 'static>> {
        let contents = fs::read_to_string(path)?;
        let mut manifest: Manifest = serde_yaml::from_str(&contents)?;
        if repo.is_some() {
            manifest.remote_url = String::from(format!("file://{}", repo.unwrap()));
        }
        return Ok(manifest);
    }
}
