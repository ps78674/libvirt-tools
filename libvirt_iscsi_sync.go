package main

import (
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/libvirt/libvirt-go"
	libvirtxml "github.com/libvirt/libvirt-go-xml"
)

var logFile string
var libvirtURI string
var iscsiAdmPath string
var iscsiAddr string
var showOnly bool

func init() {
	flag.StringVar(&logFile, "log", "/var/log/libvirt_iscsi_sync.log", "log file path")
	flag.StringVar(&libvirtURI, "uri", "qemu:///system", "Libvirt connection uri")
	flag.StringVar(&iscsiAdmPath, "path", "/usr/sbin/iscsiadm", "Path for 'iscsiadm'")
	flag.StringVar(&iscsiAddr, "addr", "", "ISCSI host address")
	flag.BoolVar(&showOnly, "show", false, "Do not create pools, just show new targets")
	flag.Parse()

	if iscsiAddr == "" {
		fmt.Printf("ISCSI host address must be set.\n\n")

		flag.Usage()
		os.Exit(1)
	}
}

func randomUUID() string {
	b := make([]byte, 20)
	rand.Read(b)

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[5:7], b[8:10], b[11:13], b[14:20])
}

func getISCSIDevicePaths(conn *libvirt.Connect) (paths []string) {
	pools, e := conn.ListAllStoragePools(libvirt.CONNECT_LIST_STORAGE_POOLS_ISCSI)
	if e != nil {
		log.Fatalln(e)
	}

	for _, p := range pools {
		desc, e := p.GetXMLDesc(0)
		if e != nil {
			log.Fatalln(e)
		}

		pool := &libvirtxml.StoragePool{}
		e = pool.Unmarshal(desc)
		if e != nil {
			log.Fatalln(e)
		}

		if pool.Source.Host[0].Name == iscsiAddr {
			paths = append(paths, pool.Source.Device[0].Path)
		}
	}
	return
}

func discoverISCSITargets() (targets []string) {
	var stderr, stdout bytes.Buffer

	iscsiAdm := exec.Command(iscsiAdmPath, "-m", "discovery", "-t", "sendtargets", "-p", iscsiAddr)
	iscsiAdm.Stderr = &stderr
	iscsiAdm.Stdout = &stdout

	e := iscsiAdm.Run()
	if e != nil {
		for _, v := range strings.Split(stderr.String(), "\n") {
			if len(v) == 0 {
				continue
			}

			log.Println(v)
		}
		log.Fatalf("iscsiadm: %s", e)
	}

	for _, line := range strings.Split(stdout.String(), "\n") {
		if len(line) > 0 {
			targets = append(targets, strings.Fields(line)[1])
		}
	}

	return
}

func getNewISCSITargets(conn *libvirt.Connect) (targets []string) {
	contains := func(slice []string, str string) bool {
		for _, e := range slice {
			if str == e {
				return true
			}
		}

		return false
	}

	for _, target := range discoverISCSITargets() {
		if !contains(getISCSIDevicePaths(conn), target) {
			targets = append(targets, target)
		}
	}

	return
}

func createLibvirtPool(conn *libvirt.Connect, target string) {
	xmlTemplate := `<pool type='iscsi'>
  <name>%s</name>
  <uuid>%s</uuid>
  <capacity unit='bytes'>0</capacity>
  <allocation unit='bytes'>0</allocation>
  <available unit='bytes'>0</available>
  <source>
    <host name=%s/>
    <device path=%s/>
  </source>
  <target>
    <path>/dev/disk/by-path</path>
  </target>
</pool>`

	name := strings.Split(target, ":")[1]
	xmlConfig := fmt.Sprintf(fmt.Sprintf("%s", xmlTemplate), name, randomUUID(), fmt.Sprintf("'%s'", iscsiAddr), fmt.Sprintf("'%s'", target))

	log.Printf("Creating pool '%s' for target '%s'\n", name, target)

	pool, e := conn.StoragePoolDefineXML(xmlConfig, 0)
	if e != nil {
		log.Fatalln(e)
	}

	pool.SetAutostart(true)
	pool.Create(0)

}

func main() {
	f, e := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		log.Fatal(e)
	}

	defer f.Close()
	log.SetOutput(f)

	conn, e := libvirt.NewConnect(libvirtURI)
	if e != nil {
		log.Fatalln(e)
	}

	defer conn.Close()

	targets := getNewISCSITargets(conn)
	if len(targets) == 0 {
		log.Println("No new targets found.")
		os.Exit(0)
	}

	for _, target := range targets {
		if showOnly {
			fmt.Println(target)
		} else {
			createLibvirtPool(conn, target)
		}
	}
}
