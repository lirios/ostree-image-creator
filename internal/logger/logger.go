// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package logger

import (
	"fmt"
	"log"
	"sync"
)

// Escape sequence for colors
var (
	colorOff    = []byte("\033[0m")
	colorRed    = []byte("\033[0;31m")
	colorGreen  = []byte("\033[0;32m")
	colorOrange = []byte("\033[0;33m")
	colorBlue   = []byte("\033[0;34m")
	colorPurple = []byte("\033[0;35m")
	colorCyan   = []byte("\033[0;36m")
	colorGray   = []byte("\033[0;37m")
)

// Global variables
var mutex sync.RWMutex
var verbose = false

// SetVerbose set the verbose flag which enables debug messages
func SetVerbose(value bool) {
	mutex.Lock()
	defer mutex.Unlock()
	verbose = value
}

// Debug print an information message
func Debug(v ...interface{}) {
	if verbose {
		log.Printf("%s%s%s", colorGray, fmt.Sprint(v...), colorOff)
	}
}

// Debugf print a formatted information message
func Debugf(format string, v ...interface{}) {
	if verbose {
		log.Printf("%s%s%s", colorGray, fmt.Sprintf(format, v...), colorOff)
	}
}

// Action print an announcement message
func Action(v ...interface{}) {
	log.Printf("\u2bc8 %s%s%s", colorBlue, fmt.Sprint(v...), colorOff)
}

// Actionf print a formatted announcement message
func Actionf(format string, v ...interface{}) {
	log.Printf("\u2bc8 %s%s%s", colorBlue, fmt.Sprintf(format, v...), colorOff)
}

// Info print an information message
func Info(v ...interface{}) {
	log.Println(v...)
}

// Infof print a formatted information message
func Infof(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// Warn print an warning message
func Warn(v ...interface{}) {
	log.Printf("%s%s%s", colorOrange, fmt.Sprint(v...), colorOff)
}

// Warnf print a formatted warning message
func Warnf(format string, v ...interface{}) {
	log.Printf("%s%s%s", colorOrange, fmt.Sprintf(format, v...), colorOff)
}

// Error print an error message
func Error(v ...interface{}) {
	log.Printf("%s%s%s", colorRed, fmt.Sprint(v...), colorOff)
}

// Errorf print a formatted error message
func Errorf(format string, v ...interface{}) {
	log.Printf("%s%s%s", colorRed, fmt.Sprintf(format, v...), colorOff)
}

// Fatal print an error message and exit
func Fatal(v ...interface{}) {
	log.Fatalf("%s%s%s", colorRed, fmt.Sprint(v...), colorOff)
}

// Fatalf print a formatted error message and exit
func Fatalf(format string, v ...interface{}) {
	log.Fatalf("%s%s%s", colorRed, fmt.Sprintf(format, v...), colorOff)
}
