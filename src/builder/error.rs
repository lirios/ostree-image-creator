/****************************************************************************
 * Copyright (C) 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 ***************************************************************************/

use crate::cmd;
use crate::ostree;

use std::fmt;
use std::io::Error as IoError;
use tera::Error as TeraError;

#[derive(Debug)]
pub enum BuildError {
    Io(IoError),
    Tera(TeraError),
    Command(cmd::CommandError),
    Ostree(ostree::OstreeError),
    Error(String),
}

impl fmt::Display for BuildError {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        match *self {
            BuildError::Io(ref err) => err.fmt(f),
            BuildError::Tera(ref err) => err.fmt(f),
            BuildError::Command(ref err) => err.fmt(f),
            BuildError::Ostree(ref err) => err.fmt(f),
            BuildError::Error(ref err) => write!(f, "{}", err),
        }
    }
}

impl From<IoError> for BuildError {
    fn from(err: IoError) -> BuildError {
        BuildError::Io(err)
    }
}

impl From<TeraError> for BuildError {
    fn from(err: TeraError) -> BuildError {
        BuildError::Tera(err)
    }
}

impl From<cmd::CommandError> for BuildError {
    fn from(err: cmd::CommandError) -> BuildError {
        BuildError::Command(err)
    }
}

impl From<ostree::OstreeError> for BuildError {
    fn from(err: ostree::OstreeError) -> BuildError {
        BuildError::Ostree(err)
    }
}
