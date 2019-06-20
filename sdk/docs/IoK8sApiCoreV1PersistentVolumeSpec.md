# IoK8sApiCoreV1PersistentVolumeSpec

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_modes** | **list[str]** | AccessModes contains all ways the volume can be mounted. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes | [optional] 
**aws_elastic_block_store** | [**IoK8sApiCoreV1AWSElasticBlockStoreVolumeSource**](IoK8sApiCoreV1AWSElasticBlockStoreVolumeSource.md) |  | [optional] 
**azure_disk** | [**IoK8sApiCoreV1AzureDiskVolumeSource**](IoK8sApiCoreV1AzureDiskVolumeSource.md) |  | [optional] 
**azure_file** | [**IoK8sApiCoreV1AzureFilePersistentVolumeSource**](IoK8sApiCoreV1AzureFilePersistentVolumeSource.md) |  | [optional] 
**capacity** | [**dict(str, IoK8sApimachineryPkgApiResourceQuantity)**](IoK8sApimachineryPkgApiResourceQuantity.md) | A description of the persistent volume&#x27;s resources and capacity. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#capacity | [optional] 
**cephfs** | [**IoK8sApiCoreV1CephFSPersistentVolumeSource**](IoK8sApiCoreV1CephFSPersistentVolumeSource.md) |  | [optional] 
**cinder** | [**IoK8sApiCoreV1CinderPersistentVolumeSource**](IoK8sApiCoreV1CinderPersistentVolumeSource.md) |  | [optional] 
**claim_ref** | [**IoK8sApiCoreV1ObjectReference**](IoK8sApiCoreV1ObjectReference.md) |  | [optional] 
**csi** | [**IoK8sApiCoreV1CSIPersistentVolumeSource**](IoK8sApiCoreV1CSIPersistentVolumeSource.md) |  | [optional] 
**fc** | [**IoK8sApiCoreV1FCVolumeSource**](IoK8sApiCoreV1FCVolumeSource.md) |  | [optional] 
**flex_volume** | [**IoK8sApiCoreV1FlexPersistentVolumeSource**](IoK8sApiCoreV1FlexPersistentVolumeSource.md) |  | [optional] 
**flocker** | [**IoK8sApiCoreV1FlockerVolumeSource**](IoK8sApiCoreV1FlockerVolumeSource.md) |  | [optional] 
**gce_persistent_disk** | [**IoK8sApiCoreV1GCEPersistentDiskVolumeSource**](IoK8sApiCoreV1GCEPersistentDiskVolumeSource.md) |  | [optional] 
**glusterfs** | [**IoK8sApiCoreV1GlusterfsVolumeSource**](IoK8sApiCoreV1GlusterfsVolumeSource.md) |  | [optional] 
**host_path** | [**IoK8sApiCoreV1HostPathVolumeSource**](IoK8sApiCoreV1HostPathVolumeSource.md) |  | [optional] 
**iscsi** | [**IoK8sApiCoreV1ISCSIPersistentVolumeSource**](IoK8sApiCoreV1ISCSIPersistentVolumeSource.md) |  | [optional] 
**local** | [**IoK8sApiCoreV1LocalVolumeSource**](IoK8sApiCoreV1LocalVolumeSource.md) |  | [optional] 
**mount_options** | **list[str]** | A list of mount options, e.g. [\&quot;ro\&quot;, \&quot;soft\&quot;]. Not validated - mount will simply fail if one is invalid. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes/#mount-options | [optional] 
**nfs** | [**IoK8sApiCoreV1NFSVolumeSource**](IoK8sApiCoreV1NFSVolumeSource.md) |  | [optional] 
**node_affinity** | [**IoK8sApiCoreV1VolumeNodeAffinity**](IoK8sApiCoreV1VolumeNodeAffinity.md) |  | [optional] 
**persistent_volume_reclaim_policy** | **str** | What happens to a persistent volume when released from its claim. Valid options are Retain (default for manually created PersistentVolumes), Delete (default for dynamically provisioned PersistentVolumes), and Recycle (deprecated). Recycle must be supported by the volume plugin underlying this PersistentVolume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#reclaiming | [optional] 
**photon_persistent_disk** | [**IoK8sApiCoreV1PhotonPersistentDiskVolumeSource**](IoK8sApiCoreV1PhotonPersistentDiskVolumeSource.md) |  | [optional] 
**portworx_volume** | [**IoK8sApiCoreV1PortworxVolumeSource**](IoK8sApiCoreV1PortworxVolumeSource.md) |  | [optional] 
**quobyte** | [**IoK8sApiCoreV1QuobyteVolumeSource**](IoK8sApiCoreV1QuobyteVolumeSource.md) |  | [optional] 
**rbd** | [**IoK8sApiCoreV1RBDPersistentVolumeSource**](IoK8sApiCoreV1RBDPersistentVolumeSource.md) |  | [optional] 
**scale_io** | [**IoK8sApiCoreV1ScaleIOPersistentVolumeSource**](IoK8sApiCoreV1ScaleIOPersistentVolumeSource.md) |  | [optional] 
**storage_class_name** | **str** | Name of StorageClass to which this persistent volume belongs. Empty value means that this volume does not belong to any StorageClass. | [optional] 
**storageos** | [**IoK8sApiCoreV1StorageOSPersistentVolumeSource**](IoK8sApiCoreV1StorageOSPersistentVolumeSource.md) |  | [optional] 
**volume_mode** | **str** | volumeMode defines if a volume is intended to be used with a formatted filesystem or to remain in raw block state. Value of Filesystem is implied when not included in spec. This is an alpha feature and may change in the future. | [optional] 
**vsphere_volume** | [**IoK8sApiCoreV1VsphereVirtualDiskVolumeSource**](IoK8sApiCoreV1VsphereVirtualDiskVolumeSource.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

