package main

import (
	"bytes"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const FUSE_ID = 0x65735546

func watcher(watchPath string) {
	for {
		waitMount(watchPath)
		fmt.Println(fmt.Sprintf("%s has been mounted", watchPath))
		time.Sleep(time.Second * time.Duration(gracePeriod))
		for {
			fmt.Println(fmt.Sprintf("Looking at %s", watchPath))
			var dirs []string
			err := filepath.Walk(watchPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					fmt.Println(fmt.Sprintf("Failed walking at %s. Is directory mounted with allow_other flag?", path))
					os.Exit(2)
				}
				if info.IsDir() {
					dirs = append(dirs, path)
				}
				return nil
			})
			if err != nil {
				os.Exit(2)
			}
			var unattended = true
			for _, dir := range dirs {
				using, err := scanDirectory(dir)
				if err != nil {
					os.Exit(2)
				}
				if using {
					unattended = false
					break
				}
			}

			if unattended {
				time.Sleep(time.Second * time.Duration(extendTime))
			} else {
				for {
					fmt.Println(fmt.Sprintf("Auto umounting %s", watchPath))
					cmd := exec.Command("fusermount", "-u", watchPath)
					var errBuf bytes.Buffer
					var outBuf bytes.Buffer
					outWriter := io.MultiWriter(os.Stdout, &outBuf)
					errWriter := io.MultiWriter(os.Stderr, &errBuf)
					cmd.Stdout = outWriter
					cmd.Stderr = errWriter
					var umountSuccess = true
					//Will fail if device is busy (like somebody cd to the mount in an tmux session), but whatever.
					if err := cmd.Run(); err != nil {
						fmt.Println(fmt.Sprintf("Failed unmounting %s! %s", watchPath, err))
						umountSuccess = false
					}
					if errBuf.String() != "" {
						fmt.Println(fmt.Sprintf("Failed unmounting %s! %s", watchPath, errBuf.String()))
						umountSuccess = false
					}
					if umountSuccess {
						break
					}
					//Infinite retires
					time.Sleep(time.Second * 5)
					fmt.Print("Retrying...")
				}
				fmt.Println(fmt.Sprintf("Successfully unmounted %s", watchPath))
				break
			}
		}
	}
}

func scanDirectory(watchPath string) (bool, error) {
	var stat unix.Statx_t
	err := unix.Statx(0, watchPath, unix.AT_STATX_SYNC_AS_STAT, unix.STATX_ATIME|unix.STATX_MTIME, &stat)
	if err != nil {
		fmt.Println(fmt.Sprintf("Failed reading %s: %s", watchPath, err))
		return false, err
	}
	atime := stat.Atime
	mtime := stat.Mtime
	accessTime := time.Unix(atime.Sec, int64(atime.Nsec))
	modifyTime := time.Unix(mtime.Sec, int64(mtime.Nsec))
	return time.Since(accessTime) > (time.Second*time.Duration(autolockTime)) || time.Since(modifyTime) > (time.Second*time.Duration(autolockTime)), nil
}

func waitMount(watchPath string) {
	for {
		var stat unix.Statfs_t
		err := unix.Statfs(watchPath, &stat)
		if err != nil {
			fmt.Println(fmt.Sprintf("Failed watching %s: %s", watchPath, err))
			os.Exit(2)
		}
		if stat.Type == FUSE_ID {
			break
		}
		time.Sleep(time.Second * time.Duration(refreshInterval))
	}
}
