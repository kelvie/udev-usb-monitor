package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/farjump/go-libudev"
)


func runcmd(cmdStr string) {
	cmd := exec.Command("/bin/sh", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Println("Running", cmdStr)
	cmd.Run()
}

func main() {
	usbvendor := flag.String("v", "0bda", "usb vendor")
	usbproduct := flag.String("p", "0411", "usb product")
	attachCmd := flag.String("attach", "echo Attached.", "command to run on attach")
	detachCmd := flag.String("detach", "echo Detached.", "command to run on detach")

	flag.Parse()

	vendornum, err := strconv.ParseUint(*usbvendor, 16, 32)
	if err != nil {
		log.Fatal("error parsing -v:", err)
	}
	productnum, err := strconv.ParseUint(*usbproduct, 16, 32)
	if err != nil {
		log.Fatal("error parsing -v:", err)
	}

	u := udev.Udev{}
	devEnum := u.NewEnumerate()
	devmon := u.NewMonitorFromNetlink("kernel")

	// Set up the monitor, to wait for detach/attach events
	err = devmon.FilterAddMatchSubsystem("usb")
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deviceCh, err := devmon.DeviceChan(ctx)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		log.Println("Starting monitor")

		product := fmt.Sprintf("%x/%x/", vendornum, productnum)

		for d := range deviceCh {
			if !strings.HasPrefix(d.Properties()["PRODUCT"], product) ||
				d.Devtype() != "usb_device" {
				continue
			}

			switch d.Action() {
			case "add":
				runcmd(*attachCmd)
			case "remove":
				runcmd(*detachCmd)
			}

		}
		wg.Done()
	}()


	// Set up the enumerator, to get the current status
	err = devEnum.AddMatchSubsystem("usb")
	if err != nil {
		log.Fatal(err)
	}
	devEnum.AddMatchIsInitialized()
	if err != nil {
		log.Fatal(err)
	}

	devEnum.AddMatchSysattr("idVendor", fmt.Sprintf("%.4x", vendornum))
	if err != nil {
		log.Fatal(err)
	}
	devEnum.AddMatchSysattr("idProduct", fmt.Sprintf("%.4x", productnum))
	if err != nil {
		log.Fatal(err)
	}

	devices, err := devEnum.Devices()
	if err != nil {
		log.Fatal(err)
	}

	// A bit of a race condition here, but it's fine.
	if len(devices) > 0 {
		runcmd(*attachCmd)
	} else {
		runcmd(*detachCmd)
	}

	wg.Wait()

}
