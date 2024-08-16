# Installing Substrate and Terraform

Most steps in the getting started guide only need to be performed once. This step is the exception. Everyone who's going to be running Substrate commands, writing Terraform code, or really interacting with AWS in any but the most superficial ways, should follow these steps.

## Substrate

### Manual installation

* <https://src-bin.com/substrate-2024.07-darwin-.html64.tar.gz>
* <https://src-bin.com/substrate-2024.07-darwin-arm64.tar.gz>
* <https://src-bin.com/substrate-2024.07-linux-.html64.tar.gz>
* <https://src-bin.com/substrate-2024.07-linux-arm64.tar.gz>

Download the tarball for your platform. Then extract it by running a command like this from your downloads directory:

```shell-session
tar xf substrate-<version>-<OS>-<ARCH>.tar.gz
```

This will create `substrate-<version>-<OS>-<ARCH>`, which contains `bin/substrate` (typically all you need), `opt/bin/` with some optional extra programs that are part of the Substrate distribution, and `src/` with the complete source code for this release of Substrate.

To simply install Substrate in `~/bin` in a single command, run a command like this:

```shell-session
tar xf substrate-<version>-<OS>-<ARCH>.tar.gz -C ~/bin --strip-components 2 substrate-<version>-<OS>-<ARCH>/bin/substrate
```

Each released _version_ and _commit_ is offered in four binary formats; choose the appropriate one for your system. _`<OS>`_ is one of “`darwin`” or “`linux`” and _`<ARCH>`_ is one of “`.html64`” or “`arm64`”.

You can install Substrate wherever you like. If `~/bin` doesn't suit you, just ensure the directory where you install it is on your `PATH`.

### Unattended installation

Installing Substrate on fleets of laptops, in EC2 instances you get from the Instance Factory, or anywhere else could be tedious if you had to follow the procedure above each time and update the version string each month so Substrate ships with an automation-friendly install method. To install the latest version, month in and month out, without having to micromanage version strings, do the following:

1. Copy `substrate-<version>-<OS>-<ARCH>.tar.gz/src/install.sh` from any Substrate release tarball
2. Arrange for `install.sh` to be distributed to your fleet of laptops, EC2 instances, or whatever other endpoints you have in mind
3. Arrange to execute `install.sh -d <dirname>` at first boot or setup time, where `<dirname>` is the name of a directory on `PATH` where Substrate will be installed.

### Upgrading

Once some version of Substrate is installed, upgrading is a simple matter of running `substrate upgrade`.

## Terraform

Substrate currently requires at least Terraform 1.5.6. (Substrate asks for Terraform to be upgraded every few releases to stay nearly current with Terraform.)

The easist way to install Terraform 1.5.6 is to run `substrate terraform install`. If the directory that contains `substrate` itself is writable, `terraform` will be placed there, too.

Alternatively, you can download [Terraform 1.5.6](https://releases.hashicorp.com/terraform/1.5.6/) from Hashicorp, with the filenames being parameterized with _`OS`_ and _`ARCH`_ the same as Substrate itself. Download and `unzip` the appropriate build. Move `terraform` into a directory on your `PATH`. (It doesn't have to be the same directory where you placed `substrate`.)
