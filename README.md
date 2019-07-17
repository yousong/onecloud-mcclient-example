Build

	export GO111MODULE=on
	go build .

Example usage

	# Initialize $OS_xx by sourcing /opt/yunionsetup/config/rc_admin
	run_() {
	  ./onecloud-mcclient-example \
		-user "$OS_USERNAME" \
		-pass "$OS_PASSWORD" \
		-project "$OS_PROJECT_NAME" \
		-domain "$OS_DOMAIN_NAME" \
		-auth-url "$OS_AUTH_URL" \
		-region "$OS_REGION_NAME" \
		"$@"
	}

	run_ \
	  -lb \
	  -lb-network vnet.2 \

	run_ \
	  --server \
	  --server-image CentOS-7.6.1810-20190430.qcow2 \
