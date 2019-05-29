Build

	export GO111MODULE=on
	go build .

Example usage

	./onecloud-mcclient-example \
	  -user sysadmin \
	  -pass xxxxxxxxxxxx \
	  -project system \
	  -domain Default \
	  -auth-url http://11.111.222.222:5000/v3 \
	  -region Yunion \
	  -lb-network vnet.2

swagger.yaml moved to [onecloud repo](ihttps://github.com/yunionio/onecloud/blob/master/docs/swagger.yaml).
It will be maintained there
