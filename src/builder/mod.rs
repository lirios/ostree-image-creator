/****************************************************************************
 * Copyright (C) 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 ***************************************************************************/

pub use self::diskbuilder::DiskBuilder;
pub use self::error::BuildError;
pub use self::livebuilder::LiveBuilder;
pub use self::manifest::ImageType;
pub use self::manifest::Manifest;

mod diskbuilder;
mod error;
mod livebuilder;
mod manifest;

pub type BuildResult = Result<(), BuildError>;

pub trait Builder {
    fn build(&self) -> BuildResult;
}
