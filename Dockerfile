FROM harbor.cloud.netease.com/curve/release:release2.5.0-beta-csi
RUN sed -i "s diskCache.diskCacheType=2 diskCache.diskCacheType=0 g" /curvefs/conf/client.conf
ADD bin/curvefs-csi-driver /usr/bin/curvefs-csi-driver
ADD https://github.com/krallin/tini/releases/download/v0.19.0/tini-amd64 /bin/tini
RUN chmod +x /bin/tini
ENTRYPOINT [ "/bin/tini", "--", "/usr/bin/curvefs-csi-driver"]
