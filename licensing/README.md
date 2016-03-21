Extract GPL source from Moby

WORK IN PROGRESS SOME ISSUES STILL

This container runs on a Pinata system and dumps out the GPL code it is running

```
docker build -t license .
docker run -it -v /etc:/hostetc -v /lib:/lib -v $PWD/output:/output license
```

TODO add kernel to this, there is now a patch to get the metadata in.
