/****************************************************************************
 * Copyright (C) 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 ***************************************************************************/

use serde::Serialize;
use tera::{Context, Result, Tera};

#[derive(Serialize)]
struct Stanza {
    name: String,
    label: String,
    kargs: String,
}

pub enum TemplateType {
    Grub,
    Syslinux,
}

pub struct Template {
    tera: tera::Tera,
    product: String,
    title: String,
    fslabel: String,
    timeout: u32,
    stanzas: Vec<Stanza>,
    vesa_kargs: String,
    memtest: bool,
}

impl Template {
    pub fn new(product: &str, title: &str, fslabel: &str, timeout: u32) -> Result<Template> {
        let mut tera = Tera::default();
        let grub = String::from_utf8_lossy(include_bytes!("files/grub.cfg"));
        let syslinux = String::from_utf8_lossy(include_bytes!("files/syslinux.cfg"));
        tera.add_raw_template("grub", &grub)?;
        tera.add_raw_template("syslinux", &syslinux)?;

        Ok(Template {
            tera: tera,
            product: product.to_string(),
            title: title.to_string(),
            fslabel: fslabel.to_string(),
            timeout: timeout,
            stanzas: vec![],
            vesa_kargs: String::from(""),
            memtest: false,
        })
    }

    pub fn add_stanza(&mut self, name: &str, label: &str, kargs: &str) {
        let stanza = Stanza {
            name: name.to_string(),
            label: label.to_string(),
            kargs: kargs.to_string(),
        };
        self.stanzas.push(stanza);
    }

    pub fn set_vesa_kargs(&mut self, kargs: &str) {
        self.vesa_kargs = kargs.to_string();
    }

    pub fn enable_memtest(&mut self, enabled: bool) {
        self.memtest = enabled;
    }

    pub fn render(&self, tt: TemplateType) -> Result<String> {
        let mut context = Context::new();
        context.insert("title", &self.title);
        context.insert("product", &self.product);
        context.insert("fslabel", &self.fslabel);
        context.insert("timeout", &self.timeout);
        context.insert("vesa_kargs", &self.vesa_kargs);
        context.insert("memtest_enabled", &self.memtest);
        context.insert("stanzas", &self.stanzas);
        self.tera.render(
            match tt {
                TemplateType::Grub => "grub",
                TemplateType::Syslinux => "syslinux",
            },
            &context,
        )
    }
}
