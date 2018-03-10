// Copyright 2018 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by the Apache License 2.0
// license that can be found in the LICENSE file.

package slog

import (
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/fcavani/slog/systemd"
)

// Prior2Sd convert the priority from slog to systemd.
func Prior2Sd(level Level) systemd.Priority {
	switch level {
	case ProtoPrio:
		return systemd.PriDebug
	case DebugPrio:
		return systemd.PriDebug
	case InfoPrio:
		return systemd.PriInfo
	case ErrorPrio:
		return systemd.PriErr
	case FatalPrio:
		return systemd.PriEmerg
	case PanicPrio:
		return systemd.PriEmerg
	case NoPrio:
		return systemd.PriWarning
	default:
		return systemd.PriNotice
	}
}

// CommitSd send to systemd journal the log entry.
func CommitSd(sl *Slog) {
	sl.Log.Timestamp = time.Now()

	if systemd.Enabled() || testing {
		buf, err := sl.Formatter(sl)
		if err != nil {
			//TODO: Give to the user a nice error message.
			println("SLOG writer failed:", err)
			return
		}

		vars := make(map[string]string, 11)

		vars["_TRANSPORT"] = "journal"
		vars["DOMAIN"] = string(sl.Log.Domain)
		vars["_PID"] = PID
		vars["_UID"] = UID
		vars["_GID"] = GID
		vars["_HOSTNAME"] = Hostname
		//vars[""] = ""

		if mid, err := systemd.MachineID(); err == nil {
			vars["_MACHINE_ID"] = mid
		}

		// TODO: Salvar o level...
		if sl.Log.DoDi {
			if fnname, file, line := debuginfo(sl.Log.DiLevel + 1); file != "" && line != "" {
				vars["CODE_FILE"] = file
				vars["CODE_LINE"] = line
				vars["CODE_FUNC"] = fnname
			}
		}

		if len(*sl.Log.Tags) > 0 {
			vars["TAGS"] = sl.Log.Tags.String()
		}

		err = sendToSd(string(buf), Prior2Sd(sl.Log.Priority), vars)
		if err != nil {
			println("SLOG writer failed:", err)
			return
		}

		return
	}

	// Fallback formatter and commiter.
	// Send the log to some file normally the os.Stdout.
	// Set slog Writter property to os.Stdout.
	if sl.Log.DoDi {
		sl.Log.file = debugInfo(sl.Log.DiLevel)
	}

	buf, err := FallbackFormater(sl)
	if err != nil {
		//TODO: Give to the user a nice error message.
		println("SLOG writer failed:", err)
		return
	}

	sl.Lck.Lock()
	_, err = sl.Writter.Write(buf)
	if err != nil {
		println("SLOG writer failed:", err)
	}
	Pool.Put(buf[:0])
	sl.Lck.Unlock()
}

// SdFormater format the mensagem for systemd journal
func SdFormater(sl *Slog) ([]byte, error) {
	buf := Pool.Get().([]byte)
	buf = append(buf, sl.Log.msg()...)
	return buf, nil
}

// FallbackFormater is called if systemd isn't avalible. Need to set Writter in
// Slog struct.
func FallbackFormater(sl *Slog) ([]byte, error) {
	buf := Pool.Get().([]byte)
	buf = append(buf, sl.Log.Domain...)
	buf = append(buf, sep...)
	FormatTime(&buf, sl.Log.Timestamp)
	buf = append(buf, sep...)
	buf = append(buf, sl.Log.Priority.Byte()...)
	buf = append(buf, sep...)
	if len(*sl.Log.Tags) > 0 {
		buf = append(buf, []byte(sl.Log.Tags.String())...)
		buf = append(buf, sepTags...)
	}
	if sl.Log.DoDi {
		buf = append(buf, []byte(sl.Log.file)...)
		buf = append(buf, sep...)
	}
	buf = append(buf, sl.Log.msg()...)
	return buf, nil
}

func debuginfo(level int) (fnname, file, line string) {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(level, pc)
	f := runtime.FuncForPC(pc[0])
	file, l := f.FileLine(pc[0])
	fnname = f.Name()
	line = strconv.Itoa(l)
	return
}

var (
	GID      string
	UID      string
	PID      string
	Hostname string

	sendToSd func(string, systemd.Priority, map[string]string) error

	testing bool
)

// Testing enable testing in an eviroment without systemd.
func Testing(t bool) {
	testing = t
	sendToSd = systemd.Send
	if t {
		sendToSd = systemd.SendMock(os.Stdout)
	}
}

func init() {
	GID = strconv.Itoa(os.Getgid())
	UID = strconv.Itoa(os.Getuid())
	PID = strconv.Itoa(os.Getpid())
	if hn, err := os.Hostname(); err == nil {
		Hostname = hn
	}
	testing = false
	sendToSd = systemd.Send
}
