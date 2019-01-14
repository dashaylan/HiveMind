## Development Test Setup

Hiveminds requires multiple nodes in order to test the implementation. This section describes how to setup your development machine to use KVM to launch additional nodes that can be used as part of the DSM. This setup assumes that you are running Ubuntu 16.04. Check this [site](https://www.cyberciti.biz/faq/how-to-use-kvm-cloud-images-on-ubuntu-linux/) for details. The exact steps are reproduced here:

### Step 1 - Install KVM and Setup Networking
First, you would need to install a few packages.

```
$ sudo apt install bridge-utils qemu-kvm libvirt-bin virtinst cpu-checker
```

Now that that is set up, it’s time to add the bridge details to the network interface configuration. To do so, open up 
the configuration as root, like in the command below.

```
$ sudo vim /etc/network/interfaces
```

In order to use the bridge, make sure that your configuration looks similar to the one below, substituting eth0 for the name of your interface (e.g. eno1)

```
# Establishing which interfaces to load at boot and establish the loopback
auto lo br0
iface lo inet loopback

# Set the existing interface to manual to keep it from interfering with the bridge via DHCP
iface eth0 inet manual

# Create the bridge and set it to DHCP.  Link it to the existing interface.
iface br0 inet dhcp
bridge_ports eth0
```

When the changed are completed, save the configuration and exit the text editor. Everything should be set for the bridge to work. Nothing else will change in terms of normal use. There will only be bridged interface available for the applications that use it. In order for the bridge to take effect, restart your system.

```
$ sudo reboot
``` 

### Step 2 - Install uvtool

```
$ sudo apt install uvtool
```

### Step 3 - Download the Ubuntu Cloud Image for KVM

To just update/grab Ubuntu 16.04 LTS (xenial/amd64) image run:

```
$ uvt-simplestreams-libvirt --verbose sync release=xenial arch=amd64
```
Sample outputs:
```
Adding: com.ubuntu.cloud:server:16.04:amd64 20180306
```
It takes a few minutes to download the image depending on your internet connection. Pass the query option to queries the local mirror:
```
$ uvt-simplestreams-libvirt query
```
Sample outputs:
```
release=xenial arch=amd64 label=release (20180306)
```
Now, you have an image for Ubuntu xenial and you create the VM.

### Step 4 - Create the SSH Keys
You need ssh keys for login into KVM VMs. Use the ssh-keygen command to create a new one if you do not have any keys at all.
```
$ ssh-keygen
```
The command will prompt you for file location and paraphrase. Just press <enter> key to accept the defaults. The key will be located in ~/.ssh

### Step 5 - Create the Ubuntu VM using cloud image

It is time to create the VM named vm1 i.e. create an Ubuntu Linux 16.04 LTS VM:
```
$ uvt-kvm create vm1
```

By default vm1 created using the following characteristics:

    1. RAM/memory : 512M
    2. Disk size: 8GiB
    3. CPU: 1 vCPU core

Check this [site](https://www.cyberciti.biz/faq/how-to-use-kvm-cloud-images-on-ubuntu-linux/) for more details on how change the defaults and also pass in other parameters.

### Step 6 - Login to the VM
There is a few ways to login to the VM. You can use the uvt-kvm command:
```
$ uvt-kvm ssh vm1 --insecure
```
where vm1 is the name of the VM that you created in the previous step. 

Alternatively, you can get the IP address of the VM using the uvt-kvm command:
```
$ uvt-kvm ip vm1
192.168.122.52
```
and then ssh to the VM as follows:
```
$ ssh ubuntu@192.168.122.52
```
### Step 6 - Delete VM
To destroy/delete your VM named vm1, run (please use the following command with care as there would be no confirmation box):
```
$ uvt-kvm destroy vm1
```

## Usage

Dependencies:
ssh: go get -u golang.org/x/crypto/ssh
TBD

## Design

### API

### Data Structures

### Shared Memory Management

### Inter-Processor Messaging

