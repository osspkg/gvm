# gvm
Golang Virtual Manager

### Install & Update Script

To **install** or **update** gvm, you should run the installation script. To do that, you may either download and run the script manually, or use the following cURL or Wget command:
```shell
curl -o- https://raw.githubusercontent.com/osspkg/gvm/master/install.sh | bash
```

### Use:

```shell
gvm install <version> # only install version
gvm default <version> # set global version and install
gvm local <version>   # set local version, generate `.gvmrc` and install
gvm update            # install last GVM version
```

### Use file `.gvmrc`
To configure the Go version for the current project, specify the desired version in the file. Example:

```text
1.22.0
```