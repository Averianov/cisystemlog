package main

import (
	l "cisystemlog"
)

func main() {
	l.CreateLogs("", "", l.ALERT, 1)
	l.CheckFileLevel()
}