package resources

import (
	"fmt"
	"github.com/kapp-staging/kapp/controller/controllers"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"regexp"
	"strings"

	//v1 "k8s.io/apiserver/pkg/apis/example/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// actual aggregation info of PVC & PV
type Volume struct {
	Name               string `json:"name"`
	Namespace          string `json:"namespace"`
	IsInUse            bool   `json:"isInUse"`                      // can be reused or not
	ComponentNamespace string `json:"componentNamespace,omitempty"` // ns of latest component using this Volume
	ComponentName      string `json:"componentName,omitempty"`      // name of latest component using this Volume
	Capacity           string `json:"capacity"`                     // size, e.g. 1Gi
	PVC                string `json:"pvc"`
	PV                 string `json:"pv"`
}

func (builder *Builder) GetPVs() ([]coreV1.PersistentVolume, error) {
	var pvList coreV1.PersistentVolumeList

	err := builder.List(&pvList)

	return pvList.Items, err
}

func (builder *Builder) GetPVCs(opts ...client.ListOption) ([]coreV1.PersistentVolumeClaim, error) {
	var pvcList coreV1.PersistentVolumeClaimList

	err := builder.List(&pvcList, opts...)

	return pvcList.Items, err
}

type volPair struct {
	pv  coreV1.PersistentVolume
	pvc coreV1.PersistentVolumeClaim
}

// 1. list all kapp pvcs
// 2. filter all available kappPVCs
// 3. separate into 2 groups: same-ns pvc & diff-ns pvc (pv reclaimType must be Retain)
// 4. resp: same-ns pvc: pvcName, diff-ns pvc: pvName
func (builder *Builder) FindAvailableVolsForSimpleWorkload(ns string) ([]Volume, error) {
	pvList, err := builder.GetPVs()
	if err != nil {
		return nil, err
	}

	pvcList, err := builder.GetPVCs()
	if err != nil {
		return nil, err
	}

	var unboundPVs []coreV1.PersistentVolume
	var boundedPVs []coreV1.PersistentVolume
	for _, pv := range pvList {
		if pv.Spec.ClaimRef == nil {
			unboundPVs = append(unboundPVs, pv)
			continue
		}

		// make sure bounded pvc still exists
		if _, exist := findPVCByClaimRef(pv.Spec.ClaimRef, pvcList); !exist {
			unboundPVs = append(unboundPVs, pv)
		} else {
			boundedPVs = append(boundedPVs, pv)
		}
	}

	var freePairs []volPair

	// find if boundedPV's pvc is in use
	for _, boundedPV := range boundedPVs {
		pvc, _ := findPVCByClaimRef(boundedPV.Spec.ClaimRef, pvcList)
		isInUse, err := builder.IsPVCInUse(pvc)
		if err != nil {
			return nil, err
		}

		if isInUse {
			continue
		}

		freePairs = append(freePairs, volPair{
			pv:  boundedPV,
			pvc: pvc,
		})
	}

	sameNsFreePairs, diffNsFreePairs := divideAccordingToNs(freePairs, ns)

	var rst []Volume
	for _, sameNsFreePair := range sameNsFreePairs {
		pvc := sameNsFreePair.pvc
		pv := sameNsFreePair.pv

		compName, compNs := GetNameAndNsOfPVOwnerComponent(pv)

		// re-use pvc
		rst = append(rst, Volume{
			Name:               pvc.Name,
			IsInUse:            false,
			ComponentNamespace: compNs,
			ComponentName:      compName,
			Capacity:           GetCapacityOfPVC(pvc),
			PVC:                sameNsFreePair.pvc.Name,
			PV:                 "",
		})
	}

	// re-use pv
	for _, diffNsFreePair := range diffNsFreePairs {
		pvc := diffNsFreePair.pvc
		pv := diffNsFreePair.pv

		compName, compNs := GetNameAndNsOfPVOwnerComponent(pv)

		// re-use pvc
		rst = append(rst, Volume{
			Name:               pvc.Name,
			IsInUse:            false,
			ComponentNamespace: compNs,
			ComponentName:      compName,
			Capacity:           GetCapacityOfPVC(pvc),
			PVC:                "",
			PV:                 pv.Name,
		})
	}

	for _, unboundPV := range unboundPVs {
		compName, compNs := GetNameAndNsOfPVOwnerComponent(unboundPV)

		// re-use pvc
		rst = append(rst, Volume{
			Name:               unboundPV.Name,
			IsInUse:            false,
			ComponentNamespace: compNs,
			ComponentName:      compName,
			Capacity:           GetCapacityOfPV(unboundPV),
			PVC:                "",
			PV:                 unboundPV.Name,
		})
	}

	return rst, nil
}

func (builder *Builder) FindAvailableVolsForSts(ns, stsName string) ([]Volume, error) {
	pvcList, err := builder.GetPVCs(client.InNamespace(ns) /*, client.MatchingLabels{controllers.KappLabelManaged: "true"}*/)
	if client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	volClaimTplName2PVCsMap := make(map[string][]coreV1.PersistentVolumeClaim)
	volClaimTplHasInUsePVC := make(map[string]interface{})

	//format of pvc generated from volClaimTemplate is: <volClaimTplName>-<stsName>-{0,1,2}
	for _, pvc := range pvcList {
		stsPVCPattern := fmt.Sprintf(`^*.-%s-[0-9]+$`, stsName)
		stsPVCRegex := regexp.MustCompile(stsPVCPattern)

		if match := stsPVCRegex.Match([]byte(pvc.Name)); !match {
			continue
		}

		idx := strings.LastIndex(pvc.Name, fmt.Sprintf("-%s-", stsName))
		volClaimTplName := pvc.Name[:idx]

		isInUse, err := builder.IsPVCInUse(pvc)
		if err != nil {
			return nil, err
		}

		if isInUse {
			volClaimTplHasInUsePVC[volClaimTplName] = true
		}

		volClaimTplName2PVCsMap[volClaimTplName] = append(volClaimTplName2PVCsMap[volClaimTplName], pvc)
	}

	// rm volClaimTmpl if any pvc belonging to it is in use
	for volClaimTpl := range volClaimTplHasInUsePVC {
		delete(volClaimTplName2PVCsMap, volClaimTpl)
	}

	rst := []Volume{}
	for freeVolClaimTpl, pvcs := range volClaimTplName2PVCsMap {
		pvc := pvcs[0]
		capacity := GetCapacityOfPVC(pvc)

		rst = append(rst, Volume{
			Name:     freeVolClaimTpl,
			IsInUse:  false,
			Capacity: capacity,
			PVC:      freeVolClaimTpl,
			PV:       "",
		})
	}

	return rst, nil
}

func GetCapacityOfPVC(pvc coreV1.PersistentVolumeClaim) string {
	var capInStr string
	if cap, exist := pvc.Spec.Resources.Requests[coreV1.ResourceStorage]; exist {
		capInStr = cap.String()
	}
	return capInStr
}

func GetCapacityOfPV(pv coreV1.PersistentVolume) string {
	var capInStr string
	if cap, exist := pv.Spec.Capacity[coreV1.ResourceStorage]; exist {
		capInStr = cap.String()
	}
	return capInStr
}

func GetNameAndNsOfPVOwnerComponent(pv coreV1.PersistentVolume) (compName, compNamespace string) {
	if v, exist := pv.Labels[controllers.KappLabelComponent]; exist {
		compName = v
	}
	if v, exist := pv.Labels[controllers.KappLabelNamespace]; exist {
		compNamespace = v
	}

	return
}

func divideAccordingToNs(pairs []volPair, ns string) (sameNs []volPair, diffNs []volPair) {
	for _, p := range pairs {
		if p.pvc.Namespace == ns {
			sameNs = append(sameNs, p)
		} else {
			diffNs = append(diffNs, p)
		}
	}

	return
}

func findPVCByClaimRef(
	ref *coreV1.ObjectReference,
	list []coreV1.PersistentVolumeClaim,
) (rst coreV1.PersistentVolumeClaim, exist bool) {
	if ref == nil {
		return
	}

	for _, pvc := range list {
		if pvc.Name == ref.Name && pvc.Namespace == ref.Namespace {
			return pvc, true
		}
	}

	return
}

func (builder *Builder) IsPVCInUse(pvc coreV1.PersistentVolumeClaim) (bool, error) {

	var podList coreV1.PodList
	err := builder.List(&podList, client.InNamespace(pvc.Namespace))
	if errors.IsNotFound(err) {
		return false, err
	}

	isInUse := isPVCInUse(pvc, podList.Items)

	return isInUse, nil
}

func isPVCInUse(pvc coreV1.PersistentVolumeClaim, podList []coreV1.Pod) bool {
	isInUse := false
	for _, pod := range podList {
		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim == nil {
				continue
			}

			if vol.PersistentVolumeClaim.ClaimName == pvc.Name {
				isInUse = true
				break
			}
		}

		if isInUse {
			break
		}
	}

	return isInUse
}
