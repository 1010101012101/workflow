# Quick Start

These steps will help you provision a Deis cluster.

## Check System Requirements

The Deis provision scripts default to a machine size which should be adequate to run Deis, but this can be customized. Please refer to the [system requirements][] for resource considerations when choosing a machine size to run Deis.

## Choose a Provider

Choose one of the following providers and deploy a new kubernetes cluster:

- [Amazon AWS](http://kubernetes.io/v1.1/docs/getting-started-guides/aws.html)
- [Google Container Engine](https://cloud.google.com/container-engine/docs/before-you-begin)
- [Vagrant](http://kubernetes.io/v1.1/docs/getting-started-guides/vagrant.html)

## Configure DNS

See [Configuring DNS][] for more information on properly setting up your DNS records with Deis.

## Install Deis Platform

Now that you've finished provisioning a cluster, please [Install the Deis Platform][install deis].

## Register a User

Once your cluster has been provisioned and the Deis Platform has been installed, you can
[install the client][client] and [register your first user][register]!

[client]: ../using-deis/installing-the-client.md
[configuring object storage]: configuring-object-storage.md
[configuring dns]: ../managing-deis/configuring-dns.md
[install deis]: installing-the-deis-platform.md
[register]: ../using-deis/registering-a-user.md
[system requirements]: system-requirements.md
