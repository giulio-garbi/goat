# Install the GoAt Virtual machine with the Eclipse Plugin

1. Install [VirtualBox](https://www.virtualbox.org/).
2. Download the [virtual machine](https://drive.google.com/file/d/1V0A_8suz3Zh4_qIKc8HKe5F8alL7VWso/view?usp=sharing).
3. Launch VirtualBox.
4. Open the `Tools` tab, then choose `Import`.
5. Click on the right icon, reach the folder where you downloaded the virtual machine, choose the file and then click `Continue`.
6. Make sure that `Import hard drives as VDI` (bottom half of the window) is unchecked.
7. Personalize the setting of the virtual machine as you want, then click `Import`.
8. Wait until the virtual machine is imported.
9. Click on the `GoAt Ubuntu` virtual machine and click on `Start`.
10. After the machine booted, you are ready to use GoAt! Visit the other tutorials to discover the features of GoAt and its Eclipse plugin:
    * The Eclipse plugin; [tutorial](plugin.md)
    * The supported infrastructures; [tutorial](infrastructure.md)


# Install GoAt API to use directly on your machine

1. Install [Google Go](https://golang.org/). We suggest to use at least version 1.9.1.
2. Install the `github.com/giulio-garbi/goat/goat` package.
    * In macOs and Linux, open a terminal and run `go get github.com/giulio-garbi/goat/goat`.
    * In Windows, make sure that the Go executables are in the `PATH` environment variable. Then open the Command Prompt and run `go get github.com/giulio-garbi/goat/goat`.
3. Define your system using the GoAt API! Visit the other tutorials to discover the features of GoAt:
    * The GoAt API in Google Go; [tutorial](library.md)
    * The supported infrastructures; [tutorial](infrastructure.md)
4. Run the files using `go run filename.go`. Remember to start the infrastructure before the components!
