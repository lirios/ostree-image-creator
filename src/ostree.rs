/****************************************************************************
 * Copyright (C) 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 ***************************************************************************/

use log::trace;
use std::path::Path;
use std::process::{Command, ExitStatus};
use std::result::Result;
use std::vec::IntoIter;

#[derive(Fail, Debug, Clone, PartialEq)]
pub enum OstreeError {
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

pub type OstreeResult<T> = Result<T, OstreeError>;

pub enum OstreeArchiveMode {
    Bare,
    Archive,
}

fn result_from_status(status: ExitStatus) -> OstreeResult<()> {
    if !status.success() {
        match status.code() {
            Some(code) => Err(OstreeError::CommandExited(code)),
            None => Err(OstreeError::ProcessTerminated),
        }
    } else {
        Ok(())
    }
}

fn string_result_from_output(output: std::process::Output, command: &str) -> OstreeResult<String> {
    if !output.status.success() {
        Err(OstreeError::CommandFailed(
            command.to_string(),
            String::from_utf8_lossy(&output.stderr).trim().to_string(),
        ))
    } else {
        Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
    }
}

fn paths_result_from_output(
    output: std::process::Output,
    command: &str,
) -> OstreeResult<IntoIter<String>> {
    if !output.status.success() {
        Err(OstreeError::CommandFailed(
            command.to_string(),
            String::from_utf8_lossy(&output.stderr).trim().to_string(),
        ))
    } else {
        let string: String = String::from_utf8_lossy(&output.stdout).trim().to_string();
        let res: Vec<String> = string.split("\x00").map(|s| s.to_string()).collect();
        Ok(res.into_iter())
    }
}

// OstreeCommand

struct OstreeCommand {
    cmd: Command,
    args: Vec<String>,
}

impl OstreeCommand {
    pub fn new() -> OstreeCommand {
        OstreeCommand {
            cmd: Command::new("ostree"),
            args: vec![String::from("ostree")],
        }
    }

    pub fn arg(&mut self, arg: &str) -> &mut OstreeCommand {
        self.cmd.arg(&arg);
        self.args.push(arg.to_string());
        self
    }

    pub fn spawn(&mut self, name: &str) -> OstreeResult<()> {
        trace!("+ {}", shell_words::join(&self.args));

        self.cmd
            .status()
            .map_err(|e| OstreeError::ExecFailed(name.to_string(), e.to_string()))
            .and_then(|status| result_from_status(status))
    }

    pub fn get_output(&mut self, name: &str) -> OstreeResult<String> {
        trace!("+ {}", shell_words::join(&self.args));

        self.cmd
            .output()
            .map_err(|e| OstreeError::ExecFailed(name.to_string(), e.to_string()))
            .and_then(|output| string_result_from_output(output, &name))
    }

    pub fn list_paths(&mut self, name: &str) -> OstreeResult<IntoIter<String>> {
        trace!("+ {}", shell_words::join(&self.args));

        self.cmd
            .output()
            .map_err(|e| OstreeError::ExecFailed(name.to_string(), e.to_string()))
            .and_then(|output| paths_result_from_output(output, &name))
    }
}

// Functions

pub fn init(repo_path: &Path, mode: OstreeArchiveMode) -> OstreeResult<()> {
    let mut cmd = OstreeCommand::new();

    cmd.arg(&format!("--repo={}", repo_path.to_string_lossy()))
        .arg("init");

    match mode {
        OstreeArchiveMode::Bare => cmd.arg("--mode=bare"),
        OstreeArchiveMode::Archive => cmd.arg("--mode=archive"),
    };

    cmd.spawn("ostree init")
}

pub fn resolve_rev(repo_path: &Path, refspec: &str) -> OstreeResult<String> {
    let mut cmd = OstreeCommand::new();

    cmd.arg(&format!("--repo={}", repo_path.to_string_lossy()))
        .arg("rev-parse")
        .arg(&refspec);

    cmd.get_output("ostree rev-parse")
}

pub fn remote_add(repo_path: &Path, osname: &str, remote_url: &str) -> OstreeResult<()> {
    let mut cmd = OstreeCommand::new();

    cmd.arg(&format!("--repo={}", repo_path.to_string_lossy()))
        .arg("remote")
        .arg("add")
        .arg("--if-not-exists")
        .arg("--no-gpg-verify")
        .arg(&osname)
        .arg(&remote_url);

    cmd.spawn("ostree remote add")
}

pub fn mirror(repo_path: &Path, osname: &str, refspec: &str) -> OstreeResult<()> {
    let mut cmd = OstreeCommand::new();

    cmd.arg(&format!("--repo={}", repo_path.to_string_lossy()))
        .arg("pull")
        .arg("--mirror")
        .arg(&format!("{}:{}", &osname, &refspec));

    cmd.spawn("ostree pull --mirror")
}

pub fn pull_local(
    srcrepo_path: &Path,
    dstrepo_path: &Path,
    refs: &Vec<String>,
) -> OstreeResult<()> {
    let mut cmd = OstreeCommand::new();

    cmd.arg(&format!("--repo={}", dstrepo_path.to_string_lossy()))
        .arg("pull-local")
        .arg("--disable-fsync")
        .arg(&srcrepo_path.to_string_lossy());
    for refspec in refs {
        cmd.arg(&refspec);
    }

    cmd.spawn("ostree pull-local")
}

pub fn list(repo_path: &Path, path: &Path, rev: &str) -> OstreeResult<IntoIter<String>> {
    let mut cmd = OstreeCommand::new();

    cmd.arg(&format!("--repo={}", repo_path.to_string_lossy()))
        .arg("ls")
        .arg("--nul-filenames-only")
        .arg(&rev)
        .arg(&path.to_string_lossy());

    cmd.list_paths("ostree ls")
}

pub fn checkout(
    repo_path: &Path,
    src_path: &Path,
    dest_path: &Path,
    rev: &str,
) -> OstreeResult<()> {
    let mut cmd = OstreeCommand::new();

    cmd.arg(&format!("--repo={}", repo_path.to_string_lossy()))
        .arg("checkout")
        .arg("--user-mode")
        .arg(&format!("--subpath={}", src_path.to_string_lossy()))
        .arg(&rev)
        .arg(&dest_path.to_string_lossy());

    cmd.spawn("ostree checkout")
}

pub fn os_init(osname: &str, sysroot_path: &Path) -> OstreeResult<()> {
    let mut cmd = OstreeCommand::new();

    cmd.arg("admin")
        .arg("os-init")
        .arg(&osname)
        .arg(&format!("--sysroot={}", sysroot_path.to_string_lossy()));

    cmd.spawn("ostree os-init")
}

pub fn deploy(osname: &str, refspec: &str, sysroot_path: &Path) -> OstreeResult<()> {
    let mut cmd = OstreeCommand::new();

    cmd.arg("admin")
        .arg("deploy")
        .arg(&refspec)
        .arg(&format!("--sysroot={}", sysroot_path.to_string_lossy()))
        .arg(&format!("--os={}", &osname));

    cmd.spawn("ostree deploy")
}
