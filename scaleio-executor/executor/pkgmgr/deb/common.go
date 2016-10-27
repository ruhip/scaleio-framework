package deb

const (
	//Environment
	aiozipCheck                = "[0-9]+ upgraded|[0-9]+ newly"
	genericInstallCheck        = "1 upgraded|1 newly"
	requiredKernelVersionCheck = "4.2.0-30-generic"

	//ScaleIO node
	mdmInstallCheck     = "mdm start/running"
	sdsInstallCheck     = "sds start/running"
	sdcInstallCheck     = "Success configuring module"
	liaInstallCheck     = "lia start/running"
	liaRestartCheck     = liaInstallCheck
	gatewayInstallCheck = "The EMC ScaleIO Gateway is running"
	gatewayRestartCheck = "scaleio-gateway start/running"

	//REX-Ray
	rexrayInstallCheck = "rexray has been installed to"

	//Isolator
	dvdcliInstallCheck = "dvdcli has been installed to"
)
