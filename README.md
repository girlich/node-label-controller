This is a vibe coded k8s controller. It checks, if a node is ready and should act as a egress gateway.
Than it sets a label. If the node becomes later un-available, the label will be removed.
In this way it is possible to have multiple nodes act as a highly available Cilium Egress Gateway.
Usually it triggers gateway changes only upon label changes and with this controller also upon
node readyness changes.
