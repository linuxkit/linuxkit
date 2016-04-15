Extract source from Moby

This container runs on a Pinata system and dumps out the source code for the
kernel and packages

```
docker build -t license .
docker run -it -v /etc:/hostetc -v /usr/:/hostusr -v /lib:/lib -v $PWD/output:/output license
```
