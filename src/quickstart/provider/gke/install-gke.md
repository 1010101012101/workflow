# Install Deis Workflow on Google Compute Engine

## Check Your Setup

First check that the `helm` command is available and the version is 0.6 or newer.

```
$ helmc --version
helmc version 0.7.0+20a7ed7
```

Ensure the `kubectl` client is installed and can connect to your Kubernetes cluster. `helm` will
use it to communicate. You can test that it is working properly by running:

```
$ helmc target
Kubernetes master is running at https://104.154.234.246
GLBCDefaultBackend is running at https://104.154.234.246/api/v1/proxy/namespaces/kube-system/services/default-http-backend
Heapster is running at https://104.154.234.246/api/v1/proxy/namespaces/kube-system/services/heapster
KubeDNS is running at https://104.154.234.246/api/v1/proxy/namespaces/kube-system/services/kube-dns
kubernetes-dashboard is running at https://104.154.234.246/api/v1/proxy/namespaces/kube-system/services/kubernetes-dashboard
```

If you see a list of targets like the one above, `helm` can communicate with the Kubernetes master.

Deis Workflow requires Kubernetes 1.2 or higher. You can test that by running:

```
$ kubectl version
Client Version: version.Info{Major:"1", Minor:"2", GitVersion:"v1.2.3", GitCommit:"882d296a99218da8f6b2a340eb0e81c69e66ecc7", GitTreeState:"clean"}
Server Version: version.Info{Major:"1", Minor:"2", GitVersion:"v1.2.3", GitCommit:"882d296a99218da8f6b2a340eb0e81c69e66ecc7", GitTreeState:"clean"}
```

## Add the Deis Chart Repository

The [Deis Chart Repository](https://github.com/deis/charts) contains everything you
need to install Deis onto your Kubernetes cluster, with a single `helmc install` command.

Run the following command to add this repository to Helm:

```
$ helmc repo add deis https://github.com/deis/charts
```

## Install Deis Workflow

Now that you have Helm installed and have added the Deis Chart Repository, install Workflow by running:

```
$ helmc fetch deis/workflow-beta3             # fetches the chart into a
                                              # local workspace
$ helmc generate -x manifests workflow-beta3  # generates various secrets
$ helmc install workflow-beta3                # injects resources into
                                              # your cluster
```

Helm will install a variety of Kubernetes resources in the `deis` namespace.
You'll need to wait for the pods that it launched to be ready. Monitor their status
by running:

```
$ kubectl get pods --namespace=deis
```

If you would like `kubectl` to automatically update as the pod states change, run (type Ctrl-C to stop the watch):
```
$ kubectl get pods --namespace=deis -w
```

Depending on the order in which the Workflow components start, you may see a few components restart. This is common during the installation process, if a component's dependencies are not yet available the component will exit and Kubernetes will automatically restart the containers.

Here, you can see that controller, builder and registry all took a few loops before there were able to star:
```
workflow [refactor-docs*]$ kubectl get pods --namespace=deis
NAME                          READY     STATUS    RESTARTS   AGE
deis-builder-miekp            1/1       Running   1          2m
deis-controller-egu7x         1/1       Running   3          2m
deis-database-ok3ev           1/1       Running   0          2m
deis-logger-fluentd-d5cb9     1/1       Running   0          2m
deis-logger-fluentd-u6azj     1/1       Running   0          2m
deis-logger-rf3z9             1/1       Running   0          2m
deis-minio-sdfyz              1/1       Running   0          2m
deis-registry-f534k           1/1       Running   4          2m
deis-router-t3qb2             1/1       Running   0          2m
deis-workflow-manager-kbpw3   1/1       Running   0          2m
```

Once you see all of the pods in the `READY` state, Deis Workflow is up and running!

Next, [configure dns](dns.md) so you can register your first user and deploy an application.
