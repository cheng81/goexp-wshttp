echo "cd-ing into src..." && \
cd ./src
echo "Building dynamichttp..." && \
6g dynamichttp.go && \
echo "Building wshttp..." && \
6g wshttp.go connwrapper.go core.go httpchannel.go wshttpreq.go && \
echo "Building wshttptest..." && \
6g wshttptest.go && 6l -o ./../wshttptest wshttptest.6 && \
echo "Removing temp files" && \
rm *.6
echo "...all done"