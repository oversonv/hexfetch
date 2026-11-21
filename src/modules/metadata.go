package modules

// Metadata describes a module for --list-modules --verbose.
type Metadata struct {
	Category    string
	Description string
	LinuxOnly   bool
	External    bool
	Network     bool
}

var moduleMetadata = map[string]Metadata{
	"title":        {Category: "system", Description: "User and host name"},
	"separator":    {Category: "system", Description: "Horizontal separator line"},
	"break":        {Category: "system", Description: "Blank line"},
	"colors":       {Category: "system", Description: "Terminal color palette"},
	"custom":       {Category: "system", Description: "Custom text from the config"},
	"command":      {Category: "system", Description: "Output of a shell command", External: true},
	"os":           {Category: "system", Description: "Operating system name"},
	"kernel":       {Category: "system", Description: "Kernel release", LinuxOnly: true},
	"uptime":       {Category: "system", Description: "System uptime", LinuxOnly: true},
	"datetime":     {Category: "system", Description: "Current date and time"},
	"locale":       {Category: "system", Description: "Locale from the environment"},
	"users":        {Category: "system", Description: "Logged-in users", LinuxOnly: true},
	"shell":        {Category: "software", Description: "Current shell and version"},
	"terminal":     {Category: "software", Description: "Terminal emulator"},
	"terminalfont": {Category: "software", Description: "Terminal font"},
	"wm":           {Category: "desktop", Description: "Window manager", LinuxOnly: true},
	"de":           {Category: "desktop", Description: "Desktop environment"},
	"initsystem":   {Category: "system", Description: "Init system", LinuxOnly: true},
	"packages":     {Category: "software", Description: "Installed packages", LinuxOnly: true},
	"theme":        {Category: "desktop", Description: "GTK theme"},
	"icons":        {Category: "desktop", Description: "Icon theme"},
	"cursor":       {Category: "desktop", Description: "Cursor theme"},
	"font":         {Category: "desktop", Description: "GTK font"},
	"host":         {Category: "hardware", Description: "Device model", LinuxOnly: true},
	"board":        {Category: "hardware", Description: "Motherboard", LinuxOnly: true},
	"bios":         {Category: "hardware", Description: "BIOS or UEFI", LinuxOnly: true},
	"product":      {Category: "hardware", Description: "System vendor and product", LinuxOnly: true},
	"chassis":      {Category: "hardware", Description: "Chassis type", LinuxOnly: true},
	"cpu":          {Category: "hardware", Description: "CPU model, cores and frequency", LinuxOnly: true},
	"cpucache":     {Category: "hardware", Description: "CPU cache sizes", LinuxOnly: true},
	"memory":       {Category: "hardware", Description: "Memory usage", LinuxOnly: true},
	"swap":         {Category: "hardware", Description: "Swap usage", LinuxOnly: true},
	"loadavg":      {Category: "system", Description: "Load average", LinuxOnly: true},
	"processes":    {Category: "system", Description: "Number of processes", LinuxOnly: true},
	"battery":      {Category: "hardware", Description: "Battery charge and status", LinuxOnly: true},
	"poweradapter": {Category: "hardware", Description: "External power adapter", LinuxOnly: true},
	"temperature":  {Category: "hardware", Description: "Hardware temperatures", LinuxOnly: true},
	"fans":         {Category: "hardware", Description: "Fan speeds", LinuxOnly: true},
	"brightness":   {Category: "hardware", Description: "Screen brightness", LinuxOnly: true},
	"sound":        {Category: "hardware", Description: "Sound card", LinuxOnly: true},
	"display":      {Category: "hardware", Description: "Connected displays", LinuxOnly: true},
	"monitor":      {Category: "hardware", Description: "Monitor model via EDID", LinuxOnly: true},
	"gpu":          {Category: "hardware", Description: "Graphics adapters", LinuxOnly: true},
	"disks":        {Category: "hardware", Description: "Mounted filesystem usage", LinuxOnly: true},
	"physicaldisk": {Category: "hardware", Description: "Physical disk models", LinuxOnly: true},
	"diskio":       {Category: "hardware", Description: "Disk I/O totals", LinuxOnly: true},
	"camera":       {Category: "hardware", Description: "Video capture devices", LinuxOnly: true},
	"localip":      {Category: "network", Description: "Local IP addresses"},
	"wifi":         {Category: "network", Description: "Wi-Fi interface, SSID and signal", LinuxOnly: true},
	"bluetooth":    {Category: "network", Description: "Bluetooth controllers", LinuxOnly: true},
	"publicip":     {Category: "network", Description: "Public IP address", Network: true},
	"dns":          {Category: "network", Description: "DNS servers", LinuxOnly: true},
	"wallpaper":    {Category: "desktop", Description: "Current wallpaper", External: true},
	"wmtheme":      {Category: "desktop", Description: "Window manager theme", External: true},
	"media":        {Category: "media", Description: "Currently playing media", External: true},
	"player":       {Category: "media", Description: "Currently playing media", External: true},
	"weather":      {Category: "media", Description: "Weather report", External: true, Network: true},
	"bootmgr":      {Category: "system", Description: "Boot manager", LinuxOnly: true},
	"cpuusage":     {Category: "hardware", Description: "CPU utilization", LinuxOnly: true},
	"gpuusage":     {Category: "hardware", Description: "GPU utilization", LinuxOnly: true},
	"chassistype":  {Category: "hardware", Description: "Extended chassis type", LinuxOnly: true},
	"osage":        {Category: "system", Description: "Approximate OS install age", LinuxOnly: true},
}

// Describe returns metadata for a registered module.
func Describe(name string) Metadata {
	return moduleMetadata[name]
}
