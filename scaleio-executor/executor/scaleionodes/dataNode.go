package scaleionodes

import (
	"time"

	log "github.com/Sirupsen/logrus"
	xplatform "github.com/dvonthenen/goxplatform"
	xplatformsys "github.com/dvonthenen/goxplatform/sys"

	common "github.com/codedellemc/scaleio-framework/scaleio-executor/executor/common"
	ubuntu14 "github.com/codedellemc/scaleio-framework/scaleio-executor/executor/pkgmgr/deb/ubuntu14"
	mgr "github.com/codedellemc/scaleio-framework/scaleio-executor/executor/pkgmgr/mgr"
	rhel7 "github.com/codedellemc/scaleio-framework/scaleio-executor/executor/pkgmgr/rpm/rhel7"
	types "github.com/codedellemc/scaleio-framework/scaleio-scheduler/types"
)

//ScaleioDataNode implementation for ScaleIO Fake Node
type ScaleioDataNode struct {
	common.ScaleioNode
	PkgMgr mgr.INodeMgr
}

//NewData generates a Data Node object
func NewData(state *types.ScaleIOFramework) *ScaleioDataNode {
	myNode := &ScaleioDataNode{}

	var pkgmgr mgr.INodeMgr
	switch xplatform.GetInstance().Sys.GetOsType() {
	case xplatformsys.OsRhel:
		pkgmgr = rhel7.NewNodeRpmRhel7Mgr(state)
	case xplatformsys.OsUbuntu:
		pkgmgr = ubuntu14.NewNodeDebUbuntu14Mgr(state)
	}
	myNode.PkgMgr = pkgmgr

	return myNode
}

//RunStateUnknown default action for StateUnknown
func (sdn *ScaleioDataNode) RunStateUnknown() {
	reboot, err := sdn.PkgMgr.EnvironmentSetup(sdn.State)
	if err != nil {
		log.Errorln("EnvironmentSetup Failed:", err)
		errState := sdn.UpdateNodeState(types.StateFatalInstall)
		if errState != nil {
			log.Errorln("Failed to signal state change:", errState)
		} else {
			log.Debugln("Signaled StateFatalInstall")
		}
		return
	}

	errState := sdn.UpdateNodeState(types.StateCleanPrereqsReboot)
	if errState != nil {
		log.Errorln("Failed to signal state change:", errState)
	} else {
		log.Debugln("Signaled StateCleanPrereqsReboot")
	}

	common.WaitForCleanPrereqsReboot(sdn)

	errState = sdn.UpdateNodeState(types.StatePrerequisitesInstalled)
	if errState != nil {
		log.Errorln("Failed to signal state change:", errState)
	} else {
		log.Debugln("Signaled StatePrerequisitesInstalled")
	}

	//requires a reboot?
	if reboot {
		log.Infoln("Reboot required before StatePrerequisitesInstalled!")

		if sdn.State.Debug {
			log.Infoln("Skipping the reboot since Debug is TRUE")
		} else {
			ip1, err1 := xplatform.GetInstance().Nw.AutoDiscoverIP()
			ip2, err2 := sdn.Config.ParseIPFromRestURI()

			if err1 == nil && err2 == nil && ip1 == ip2 {
				log.Infoln("Delay reboot host running the Scheduler")
				time.Sleep(time.Duration(common.DelayForRebootInSeconds) * time.Second)
			}

			rebootErr := xplatform.GetInstance().Run.Command(common.RebootCmdline, common.RebootCheck, "")
			if rebootErr != nil {
				log.Errorln("Install Kernel Failed:", rebootErr)
			}

			time.Sleep(time.Duration(common.WaitForRebootInSeconds) * time.Second)
		}
	} else {
		log.Infoln("No need to reboot while installing prerequisites")
	}
}

//RunStatePrerequisitesInstalled default action for StatePrerequisitesInstalled
func (sdn *ScaleioDataNode) RunStatePrerequisitesInstalled() {
	err := sdn.PkgMgr.NodeSetup(sdn.State)
	if err != nil {
		log.Errorln("NodeSetup Failed:", err)
		errState := sdn.UpdateNodeState(types.StateFatalInstall)
		if errState != nil {
			log.Errorln("Failed to signal state change:", errState)
		} else {
			log.Debugln("Signaled StateFatalInstall")
		}
		return
	}

	errState := sdn.UpdateNodeState(types.StateInstallRexRay)
	if errState != nil {
		log.Errorln("Failed to signal state change:", errState)
	} else {
		log.Debugln("Signaled StateInstallRexRay")
	}
}

//RunStateInstallRexRay default action for StateInstallRexRay
func (sdn *ScaleioDataNode) RunStateInstallRexRay() {
	if sdn.State.ScaleIO.Preconfig.PreConfigEnabled {
		log.Debugln("Pre-Config is enabled skipping wait for Cluster Initialization")
	} else {
		//we need to wait because without the gateway, the rexray service restart
		//will fail
		common.WaitForClusterInitializeFinish(sdn)
	}

	reboot, err := sdn.PkgMgr.RexraySetup(sdn.State)
	if err != nil {
		log.Errorln("REX-Ray setup Failed:", err)
		errState := sdn.UpdateNodeState(types.StateFatalInstall)
		if errState != nil {
			log.Errorln("Failed to signal state change:", errState)
		} else {
			log.Debugln("Signaled StateFatalInstall")
		}
		return
	}

	err = sdn.PkgMgr.SetupIsolator(sdn.State)
	if err != nil {
		log.Errorln("Mesos Isolator setup Failed:", err)
		errState := sdn.UpdateNodeState(types.StateFatalInstall)
		if errState != nil {
			log.Errorln("Failed to signal state change:", errState)
		} else {
			log.Debugln("Signaled StateFatalInstall")
		}
		return
	}

	errState := sdn.UpdateNodeState(types.StateCleanInstallReboot)
	if errState != nil {
		log.Errorln("Failed to signal state change:", errState)
	} else {
		log.Debugln("Signaled StateCleanInstallReboot")
	}

	common.WaitForCleanInstallReboot(sdn)

	//requires a reboot?
	if reboot {
		log.Infoln("Reboot required before StateFinishInstall!")
		log.Debugln("reboot:", reboot)

		errState = sdn.UpdateNodeState(types.StateSystemReboot)
		if errState != nil {
			log.Errorln("Failed to signal state change:", errState)
		} else {
			log.Debugln("Signaled StateSystemReboot")
		}

		if sdn.State.Debug {
			log.Infoln("Skipping the reboot since Debug is TRUE")
		} else {
			ip1, err1 := xplatform.GetInstance().Nw.AutoDiscoverIP()
			ip2, err2 := sdn.Config.ParseIPFromRestURI()

			if err1 == nil && err2 == nil && ip1 == ip2 {
				log.Infoln("Delay reboot host running the Scheduler")
				time.Sleep(time.Duration(common.DelayForRebootInSeconds) * time.Second)
			}

			rebootErr := xplatform.GetInstance().Run.Command(common.RebootCmdline, common.RebootCheck, "")
			if rebootErr != nil {
				log.Errorln("Install Kernel Failed:", rebootErr)
			}

			time.Sleep(time.Duration(common.WaitForRebootInSeconds) * time.Second)
		}
	} else {
		log.Infoln("No need to reboot while installing REX-Ray")

		errState = sdn.UpdateNodeState(types.StateFinishInstall)
		if errState != nil {
			log.Errorln("Failed to signal state change:", errState)
		} else {
			log.Debugln("Signaled StateFinishInstall")
		}
	}
}

//RunStateSystemReboot default action for StateSystemReboot
func (sdn *ScaleioDataNode) RunStateSystemReboot() {
	errState := sdn.UpdateNodeState(types.StateFinishInstall)
	if errState != nil {
		log.Errorln("Failed to signal state change:", errState)
	} else {
		log.Debugln("Signaled StateFinishInstall")
	}
}

//RunStateFinishInstall default action for StateFinishInstall
func (sdn *ScaleioDataNode) RunStateFinishInstall() {
	log.Debugln("In StateFinishInstall. Wait for", common.PollForChangesInSeconds,
		"seconds for changes in the cluster.")
	time.Sleep(time.Duration(common.PollForChangesInSeconds) * time.Second)

	//TODO temporary until libkv
	sdn.LeaveMarkerFileForConfigured()
}

//RunStateUpgradeCluster default action for StateUpgradeCluster
func (sdn *ScaleioDataNode) RunStateUpgradeCluster() {
	log.Debugln("In StateUpgradeCluster. Do nothing.")
	//TODO process the upgrade here
}
