# Tailscale

## On Kubernetes

```
helm repo add tailscale https://pkgs.tailscale.com/helmcharts
```

Follow the [documentation](https://tailscale.com/kb/1236/kubernetes-operator) to install on Kubernetes. 

I needed to add to the proxy:

``` yaml
requests:
  limits:
    squat.ai/tun: "1"
```

To do this, I need to add a `ProxyClass` resource. This is in the `system/tailscale-proxy-class.yaml` file.


## For Talos

This video gives a full walkthrough: [How to install Tailscale on Talos Linux](https://www.youtube.com/watch?v=wjDtoe-CYoI)
