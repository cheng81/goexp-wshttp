E=$1
C=${1}g
L=${1}l

echo "cd-ing into src..." && \
cd ./src
echo "Building dynamichttp..." && \
$C dynamichttp.go && \
echo "Building wshttp..." && \
$C wshttp.go connwrapper.go core.go httpchannel.go wshttpreq.go && \
echo "Building wshttptest..." && \
$C wshttptest.go && $L -o ./../wshttptest wshttptest.$E && \
echo "Removing temp files" && \
rm *.$E
echo "...all done"