/****************************************************************************
 * Copyright (C) 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 ***************************************************************************/

use log::trace;
use shell_words;
use std::io::Error as IoError;
use std::path::Path;
use std::process::{Command, ExitStatus, Output, Stdio};
use std::result::Result;

#[derive(Fail, Debug, Clone, PartialEq)]
pub enum CommandError {
    #[fail(display = "Command {} failed to start: {}", _0, _1)]
    ExecFailed(String, String),
    #[fail(display = "Command exited with: {}", _0)]
    CommandExited(i32),
    #[fail(display = "Process terminated by signal")]
    ProcessTerminated,
    #[fail(display = "Command {} exited unsucessfully with stderr: {}", _0, _1)]
    CommandFailed(String, String),
    #[fail(display = "Internal Error: {}", _0)]
    InternalError(String),
}

impl From<IoError> for CommandError {
    fn from(err: IoError) -> CommandError {
        CommandError::InternalError(err.to_string())
    }
}

pub type CommandResult<T> = Result<T, CommandError>;

fn result_from_status(status: ExitStatus) -> CommandResult<()> {
    if !status.success() {
        match status.code() {
            Some(code) => Err(CommandError::CommandExited(code)),
            None => Err(CommandError::ProcessTerminated)
        }
    } else {
        Ok(())
    }
}

fn result_from_output(output: Output, command: &str) -> CommandResult<()> {
    if !output.status.success() {
        Err(CommandError::CommandFailed(
            command.to_string(),
            String::from_utf8_lossy(&output.stderr).trim().to_string(),
        ))
    } else {
        Ok(())
    }
}

fn string_result_from_output(output: Output, command: &str) -> CommandResult<String> {
    if !output.status.success() {
        Err(CommandError::CommandFailed(
            command.to_string(),
            String::from_utf8_lossy(&output.stderr).trim().to_string(),
        ))
    } else {
        Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
    }
}

pub fn run(args: &[&str]) -> CommandResult<()> {
    trace!("+ {}", shell_words::join(args));

    Command::new(&args[0])
        .args(&args[1..])
        .status()
        .map_err(|e| CommandError::ExecFailed(args[0].to_string(), e.to_string()))
        .and_then(|status| result_from_status(status))
}

pub fn run_with_cwd(args: &[&str], cwd: &Path) -> CommandResult<()> {
    trace!("+ {}", shell_words::join(args));

    Command::new(&args[0])
        .args(&args[1..])
        .current_dir(&cwd)
        .status()
        .map_err(|e| CommandError::ExecFailed(args[0].to_string(), e.to_string()))
        .and_then(|status| result_from_status(status))
}

pub fn check_output(args: &[&str]) -> CommandResult<String> {
    trace!("+ {}", shell_words::join(args));

    Command::new(&args[0])
        .args(&args[1..])
        .output()
        .map_err(|e| CommandError::ExecFailed(args[0].to_string(), e.to_string()))
        .and_then(|output| string_result_from_output(output, &args[0]))
}

pub fn no_output(args: &[&str]) -> CommandResult<()> {
    trace!("+ {}", shell_words::join(args));

    Command::new(&args[0])
        .args(&args[1..])
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .output()
        .map_err(|e| CommandError::ExecFailed(args[0].to_string(), e.to_string()))
        .and_then(|output| result_from_output(output, &args[0]))
}
