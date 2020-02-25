# 🚲 terraform-launch

> Terraform training wheels for DC/OS Users

This project helps you getting started with terraform-based deployments if you are new to terraform, yet experienced DC/OS users. This tool is **not** a replacement to terraform, but it rather helps you build some expertise around the tool.

## Description

The `terraform-launch` is a pass-through  wrapper around terraform. This means that they are 100% command-line compatible with `terraform`.

It only provides the following _additional_ functionality:

* Makes sure that the `terraform` version used is exactly 0.11.x
* Makes sure that there is an `ssh-agent` running and the correct keys are installed
* Makes sure you have the correct AWS credentials for launching a cluster
* Performs sanity checks to the terraform configuration files and provides helpful messages
* It provides some additional commands to create terraform files from scratch.

## Usage

### Deploy a cluster on AWS

1. Create an empty directory and chdir into it
2. Run `terraform-launch add-aws-cluster` to create a DC/OS cluster deployment file
3. Open `cluster-aws.tf` and adjust the parameters to your needs
4. Deploy your cluster doing:
    ```sh
    terraform-launch plan -out=plan.out
    terraform-launch apply plan.out
    ```

5. To destroy the cluster do:
    ```sh
    terraform-launch destroy
    ```

### Deploy a DC/OS package from universe

> ℹ️ You can run this command multiple times to deploy multiple services.
> You can even run it after you have added a DC/OS cluster definition; in
> which case both a cluster and your service will be provisioned for you.

1. Run `terraform-launch add-package -package=<name>` to create a service deployment file. The `<name>` is the package name as found in DC/OS universe. 
2. Open `service-<name>.tf` and adjust it to your needs 
3. Deploy your package doing:
    ```sh
    terraform-launch plan -out=plan.out
    terraform-launch apply plan.out
    ```

4. To remove the package do:
    ```sh
    terraform-launch destroy
    ```

### As `dcos-launch` replacement

> ℹ️ This is an experimental feature, please report bugs

The `terraform-launch` utility can be used as an interim, drop-in replacement to `dcos-launch` in order to help you migrate to Universal Installer.

It can parse configuration YAMLs and create the respective terraform definition, enabling pure terraform interfacing from this point onwards.

<table>
    <tr>
        <th>
            Replace
        </th>
        <th>
            With
        </th>
    </tr>
    <tr>
        <td>
            <pre>
                dcos-launch -c cluster.yaml create
                dcos-launch wait
            </pre>
        </td>
        <td>
            <pre>
                terraform-launch import-cluster cluster.yaml
                terraform-launch plan -out=plan.out
                terraform-launch apply plan.out
            </pre>
        </td>
    </tr>
    <tr>
        <td>
            <pre>
                dcos-launch describe
            </pre>
        </td>
        <td>
            <pre>
                terraform-launch output -json
            </pre>
        </td>
    </tr>
    <tr>
        <td>
            <pre>
                dcos-launch destroy
            </pre>
        </td>
        <td>
            <pre>
                terraform-launch destroy -auto-approve
            </pre>
        </td>
    </tr>
</table>
