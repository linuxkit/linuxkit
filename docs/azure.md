# Using LinuxKit on Azure

This is a quick guide to running VMs based on LinuxKit images on Azure. Please note that since we are building very minimal operating systems, without adding the [Azure Linux Agent](https://github.com/Azure/WALinuxAgent), after creating the VM, the portal will report that the creation failed. If you created the VHD properly, you will still be able to SSH into the machine.

When running `linuxkit run azure`, the image you created using `moby build` will be uploaded to Azure in a resource group, and a VM will be created, along with the necessary resources (virtual network, subnet, storage account, network security group, public IP address).

Since Azure does not offer access to the serial output of the VM, you need to have SSH access to the machine in order to attach to it. Please see the example below.

## Setup

First of all, you need to authenticate LinuxKit with your Azure subscription. For this, you need to set the following environment variables in your bash sesssion:

```
// AZURE_TENANT_ID: contains your Azure Active Directory tenant ID or domain
// AZURE_SUBSCRIPTION_ID: contains your Azure Subscription ID
// AZURE_CLIENT_ID: contains your Azure Active Directory Application Client ID
// AZURE_CLIENT_SECRET: contains your Azure Active Directory Application Secret
```

- you can [get the Azure tenant ID following the instructions here](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal#get-tenant-id)
- to get the subscription ID, log in to the Azure portal, then go to Subscriptions
- then, you need to [create an Azure Active Directory application and retrieve its ID and secret following the instructions here](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal#create-an-azure-active-directory-application)

Then, set the environment variables:

```
export AZURE_TENANT_ID=<your_tenant_id>
export AZURE_SUBSCRIPTION_ID=<your_subscription_id>
export AZURE_CLIENT_ID=<your_client_id>
export AZURE_CLIENT_SECRET=<your_client_secret>
```

Now you should be ready to deploy resources using the LinuxKit command line.

## Build an image

> This is a preliminary example image with SSHD and Docker services. In the future, there will be an `azure.yml` file in the `examples` directory

Create a new `dev.yml` file [based on the Azure example](../examples/azure.yml), generate a new SSH key and add it in the `yml`, then `moby build dev.yml`.


This will output a `dev.vhd` image that you will deploy on Azure using `linuxkit`.

## Create a new Azure VM based on the image

Now that we have a `dev.vhd` image, we can deploy a new VM to Azure based on it.

`linuxkit run azure --resourceGroupName <resource-goup-name> --accountName <storageaccountname> --location westeurope <path-to-your-dev.vhd>`

Sample output of the command:

```
Creating resource group in westeurope
Creating storage account in westeurope, resource group linuxkit-azure2
2017/05/30 12:51:49 Using default parallelism [8*NumCPU] : 16
Computing MD5 Checksum..
 Completed:  98% RemainingTime: 00h:00m:00s Throughput: 272 MB/sec
Detecting empty ranges..
 Empty ranges : 449/513
Effective upload size: 126.00 MB (from 1024.00 MB originally)
Uploading the VHD..
 Completed: 100% [    126.00 MB] RemainingTime: 00h:00m:00s Throughput: 0 Mb/sec      
Upload completed

OS Image uploaded at https://linuxkitazure2.blob.core.windows.net/linuxkitcontainer/linuxkitimage.vhd
Creating virtual network in resource group linuxkit-azure2, in westeurope
Creating subnet linuxkitsubnet584 in resource group linuxkit-azure2, within virtual network linuxkitvirtualnetwork916
Creating public IP Address in resource group linuxkit-azure2, with name publicip368
publicip368
Started deployment of virtual machine linuxkitvm493 in resource group linuxkit-azure2
Creating virtual machine in resource group linuxkit-azure2, with name linuxkitvm493, in location westeurope
NOTE:  Since you created a minimal VM without the Azure Linux Agent, the portal will notify you that the deployment failed. After around 50 seconds try connecting to the VM

ssh -i path-to-key root@publicip368.westeurope.cloudapp.azure.com

```

After around 50 seconds, try to SSH into the machine (if you added the SSHD service to the image).


## Limitations, workarounds and work in progress

- as stated before, since this image does not contain the Azure Linux Agent, the Azure Portal will report the creation as failed
- the main workaround is the way the VHD is uploaded, specifically by using a Docker container based on [Azure VHD Utils](https://github.com/Microsoft/azure-vhd-utils). This is mainly because the tool manages fast and efficient uploads, leveraging parallelism
- there is work in progress to specify what ports to open on the VM (more specifically on a network security group)
