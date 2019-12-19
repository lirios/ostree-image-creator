/****************************************************************************
 * Copyright (C) 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 ***************************************************************************/

pub use self::diskcreator::DiskCreator;
pub use self::error::BuildError;
pub use self::livecreator::LiveCreator;
pub use self::manifest::ImageType;
pub use self::manifest::Manifest;

mod diskcreator;
mod error;
mod livecreator;
mod manifest;

pub type BuildResult = Result<(), BuildError>;

pub trait Creator {
    fn build(&self) -> BuildResult;
}
