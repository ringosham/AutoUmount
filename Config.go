package main

type Config struct {
	GracePeriod     int      `toml:"grace_period"`
	RefreshInterval int      `toml:"refresh_interval"`
	AutoLockTime    int      `toml:"autolock_time"`
	ExtendTime      int      `toml:"extend_time"`
	WatchPaths      []string `toml:"watch_paths"`
}
