## deploy controller
```
make docker-build  # build container image
make install       # apply CRD
make deploy        # deploy manifests 
```

## deploy plugin
```
cp ./plugin/kubectl-loadtest /usr/local/bin/
```
