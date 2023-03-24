# Changelog

## [v1.6.0](https://github.com/kubeflow/training-operator/tree/v1.6.0) (2023-03-21)

Note: Since scheduler-plugins has changed API from `sigs.k8s.io` with the `x-k8s.io`, future releases of training operator(v1.7+) will not support scheduler-plugins v0.24.x or lower. Related: [\#1773](https://github.com/kubeflow/training-operator/pull/1773) 

Note: Latest [Python SDK 1.6 version](https://pypi.org/project/kubeflow-training/1.6.0/) does not support earlier training operator versions. The minimum training operator version required is v1.6.0 release. Related: [\#1702](https://github.com/kubeflow/training-operator/pull/1702)


**New Features**
- Support for k8s v1.25 in CI [\#1684](https://github.com/kubeflow/training-operator/pull/1684) ([johnugeorge](https://github.com/johnugeorge))
- HPA support for PyTorch Elastic [\#1701](https://github.com/kubeflow/training-operator/pull/1701) ([johnugeorge](https://github.com/johnugeorge))
- Adopting coschduling plugin [\#1724](https://github.com/kubeflow/training-operator/pull/1724) ([tenzen-y](https://github.com/tenzen-y))
- Support for Paddlepaddle [\#1675](https://github.com/kubeflow/training-operator/pull/1675) ([kuizhiqing](https://github.com/kuizhiqing))
- Create TFJob and PyTorchJob from Function APIs in the Training SDK [\#1659](https://github.com/kubeflow/training-operator/pull/1659) ([andreyvelich](https://github.com/andreyvelich))
- \[SDK\] Use Training Client without Kube Config [\#1740](https://github.com/kubeflow/training-operator/pull/1740) ([andreyvelich](https://github.com/andreyvelich))
- \[SDK\] Create Unify Training Client [\#1719](https://github.com/kubeflow/training-operator/pull/1719) ([andreyvelich](https://github.com/andreyvelich))


**Bug fixes**
- [SDK] pod has no metadata attr anymore in the get\_job\_logs\(\) … [\#1760](https://github.com/kubeflow/training-operator/pull/1760) ([yaobaiwei](https://github.com/yaobaiwei))
- Add PodGroup as controller watch source [\#1666](https://github.com/kubeflow/training-operator/pull/1666) ([ggaaooppeenngg](https://github.com/ggaaooppeenngg))
- fix infinite loop in init-pytorch container [\#1756](https://github.com/kubeflow/training-operator/pull/1756) ([kidddddddddddddddddddddd](https://github.com/kidddddddddddddddddddddd))
- Fix the success condition of the job in PyTorchJob's Elastic mode. [\#1752](https://github.com/kubeflow/training-operator/pull/1752) ([Syulin7](https://github.com/Syulin7))
- Fix XGBoost conditions bug [\#1737](https://github.com/kubeflow/training-operator/pull/1737) ([tenzen-y](https://github.com/tenzen-y))
- To fix scaledown error, upgrade PyTorch version to v1.13.1 in echo example [\#1733](https://github.com/kubeflow/training-operator/pull/1733) ([tenzen-y](https://github.com/tenzen-y))
- fix: support MxNet single host training when update mxJob status [\#1644](https://github.com/kubeflow/training-operator/pull/1644) ([PeterChg](https://github.com/PeterChg))
- fix: fix mxnet failed to update StartTime and CompletionTime [\#1643](https://github.com/kubeflow/training-operator/pull/1643) ([PeterChg](https://github.com/PeterChg))
- Fix the default LeaderElectionID and make it an argument [\#1639](https://github.com/kubeflow/training-operator/pull/1639) ([goyalankit](https://github.com/goyalankit))
- fix: fix wrong parameter for resolveControllerRef [\#1583](https://github.com/kubeflow/training-operator/pull/1583) ([fighterhit](https://github.com/fighterhit))
- fix: tfjob with restartPolicy=ExitCode not work [\#1562](https://github.com/kubeflow/training-operator/pull/1562) ([cheimu](https://github.com/cheimu))
- fix: Mac M1 compatible Dockerfile and bump TF version [\#1700](https://github.com/kubeflow/training-operator/pull/1700) ([terrytangyuan](https://github.com/terrytangyuan))
- Fix status lost [\#1697](https://github.com/kubeflow/training-operator/pull/1697) ([ggaaooppeenngg](https://github.com/ggaaooppeenngg))
- handle all restart policies [\#1649](https://github.com/kubeflow/training-operator/pull/1649) ([abin-thomas-by](https://github.com/abin-thomas-by))
- \[chore\] fix typo [\#1648](https://github.com/kubeflow/training-operator/pull/1648) ([tenzen-y](https://github.com/tenzen-y))

**Misc**
- Add validation for verifying that the CustomJob \(e.g., TFJob\) name meets DNS1035 [\#1748](https://github.com/kubeflow/training-operator/pull/1748) ([tenzen-y](https://github.com/tenzen-y))
- Configure controller worker threads [\#1707](https://github.com/kubeflow/training-operator/pull/1707) ([HeGaoYuan](https://github.com/HeGaoYuan))
- Validation Spec consistency [\#1705](https://github.com/kubeflow/training-operator/pull/1705) ([HeGaoYuan](https://github.com/HeGaoYuan))
- \[SDK\] Remove Final Keyword from constants [\#1676](https://github.com/kubeflow/training-operator/pull/1676) ([andreyvelich](https://github.com/andreyvelich))
- Fix Python installation in CI [\#1759](https://github.com/kubeflow/training-operator/pull/1759) ([tenzen-y](https://github.com/tenzen-y))
- Update mpijob\_controller.go [\#1755](https://github.com/kubeflow/training-operator/pull/1755) ([yshalabi](https://github.com/yshalabi))
- Set the default value of CleanPodPolicy to None [\#1754](https://github.com/kubeflow/training-operator/pull/1754) ([Syulin7](https://github.com/Syulin7))
- Update join Slack link [\#1750](https://github.com/kubeflow/training-operator/pull/1750) ([Syulin7](https://github.com/Syulin7))
- Update latest operator image [\#1742](https://github.com/kubeflow/training-operator/pull/1742) ([johnugeorge](https://github.com/johnugeorge))
- Run E2E with various Python versions to verify Python SDK [\#1741](https://github.com/kubeflow/training-operator/pull/1741) ([tenzen-y](https://github.com/tenzen-y))
- Add Yuki to reviewer group [\#1739](https://github.com/kubeflow/training-operator/pull/1739) ([johnugeorge](https://github.com/johnugeorge))
- Trim down CRD descriptions [\#1735](https://github.com/kubeflow/training-operator/pull/1735) ([tenzen-y](https://github.com/tenzen-y))
- Add CI to build example images [\#1731](https://github.com/kubeflow/training-operator/pull/1731) ([tenzen-y](https://github.com/tenzen-y))
- Fix predicates of paddlepaddle-controller for scheduling.volcano.sh/v1beta1 PodGroup [\#1730](https://github.com/kubeflow/training-operator/pull/1730) ([tenzen-y](https://github.com/tenzen-y))
- Fix indents on examples for tensorflow [\#1726](https://github.com/kubeflow/training-operator/pull/1726) ([tenzen-y](https://github.com/tenzen-y))
- docs: Update Kubernetes requirement and version matrix [\#1721](https://github.com/kubeflow/training-operator/pull/1721) ([terrytangyuan](https://github.com/terrytangyuan))
- chore: Update the use of MultiWorkerMirroredStrategy in TF [\#1715](https://github.com/kubeflow/training-operator/pull/1715) ([terrytangyuan](https://github.com/terrytangyuan))
- Removing deprecated Job Labels [\#1702](https://github.com/kubeflow/training-operator/pull/1702) ([johnugeorge](https://github.com/johnugeorge))
- Bump certifi from 2022.9.14 to 2022.12.7 in /py/kubeflow/tf\_operator [\#1699](https://github.com/kubeflow/training-operator/pull/1699) ([dependabot[bot]](https://github.com/apps/dependabot))
- Add myself to reviewer. [\#1689](https://github.com/kubeflow/training-operator/pull/1689) ([kuizhiqing](https://github.com/kuizhiqing))
- Upgrade the envtest version [\#1687](https://github.com/kubeflow/training-operator/pull/1687) ([tenzen-y](https://github.com/tenzen-y))
- \[chore\] Upgrade some actions version [\#1686](https://github.com/kubeflow/training-operator/pull/1686) ([tenzen-y](https://github.com/tenzen-y))
- Upgrade Golangci-lint [\#1685](https://github.com/kubeflow/training-operator/pull/1685) ([johnugeorge](https://github.com/johnugeorge))
- Make a generic logger instead of the nil logger on dependent update [\#1680](https://github.com/kubeflow/training-operator/pull/1680) ([ggaaooppeenngg](https://github.com/ggaaooppeenngg))
- Bump protobuf from 3.8.0 to 3.18.3 in /py/kubeflow/tf\_operator [\#1669](https://github.com/kubeflow/training-operator/pull/1669) ([dependabot[bot]](https://github.com/apps/dependabot))
- Removed GOARCH dependency for multiarch support [\#1674](https://github.com/kubeflow/training-operator/pull/1674) ([pranavpandit1](https://github.com/pranavpandit1))
- Update deployment.yaml [\#1668](https://github.com/kubeflow/training-operator/pull/1668) ([OmriShiv](https://github.com/OmriShiv))
- Upgrade Go version to v1.19 [\#1663](https://github.com/kubeflow/training-operator/pull/1663) ([tenzen-y](https://github.com/tenzen-y))
- Upgrade kubernetes versoin for test [\#1667](https://github.com/kubeflow/training-operator/pull/1667) ([tenzen-y](https://github.com/tenzen-y))
- Adding support for linux/ppc64le in github actions for training-operator [\#1692](https://github.com/kubeflow/training-operator/pull/1692) ([amitmukati-2604](https://github.com/amitmukati-2604))
- style: Refine name and signature of 2 replicaName functions [\#1660](https://github.com/kubeflow/training-operator/pull/1660) ([houz42](https://github.com/houz42))
- Update training operator sdk version to 1.5.0 [\#1651](https://github.com/kubeflow/training-operator/pull/1651) ([johnugeorge](https://github.com/johnugeorge))
- Add finalizers to cluster-role [\#1646](https://github.com/kubeflow/training-operator/pull/1646) ([ArangoGutierrez](https://github.com/ArangoGutierrez))
- Update the cmd to support MPI operator in ReadME [\#1656](https://github.com/kubeflow/training-operator/pull/1656) ([denkensk](https://github.com/denkensk))

**Closed issues:**

- The default value for CleanPodPolicy is inconsistent. [\#1753](https://github.com/kubeflow/training-operator/issues/1753)
- HPA support for PyTorch Elastic  [\#1751](https://github.com/kubeflow/training-operator/issues/1751)
- Bug: allowance of non DNS-1035 compliant PyTorchJob names results in service creation failures and missing state [\#1745](https://github.com/kubeflow/training-operator/issues/1745)
- paddle-operator can not get podgroup status\(inqueue\) with volcano when enable gang  [\#1729](https://github.com/kubeflow/training-operator/issues/1729)
- \*job API\(master\) cannot compatible with old job [\#1725](https://github.com/kubeflow/training-operator/issues/1725)
- Support coscheduling plugin [\#1722](https://github.com/kubeflow/training-operator/issues/1722)
- Number of worker threads used by the controller can't be configured [\#1706](https://github.com/kubeflow/training-operator/issues/1706)
- Conformance: Training tests [\#1698](https://github.com/kubeflow/training-operator/issues/1698)
- PyTorch and MPI Operator pulls hardcoded initContainer [\#1696](https://github.com/kubeflow/training-operator/issues/1696)
- PaddlePaddle Training: why can't find pods  [\#1694](https://github.com/kubeflow/training-operator/issues/1694)
- Training-operator pod CrashLoopBackOff in K8s v1.23.6 with kubeflow1.6.1 [\#1693](https://github.com/kubeflow/training-operator/issues/1693)
- \[SDK\] Create unify client for all Training Job types [\#1691](https://github.com/kubeflow/training-operator/issues/1691)
- Support Kubernetes v1.25 [\#1682](https://github.com/kubeflow/training-operator/issues/1682)
- panic happened when add podgroup watch [\#1679](https://github.com/kubeflow/training-operator/issues/1679)
- OnDependentUpdateFunc for Job will panic when enable volcano scheduler [\#1678](https://github.com/kubeflow/training-operator/issues/1678)
- There is no clusterrole of "MPI Jobs" in kubeflow 1.5. [\#1670](https://github.com/kubeflow/training-operator/issues/1670)
- Change Kubernetes version for test [\#1665](https://github.com/kubeflow/training-operator/issues/1665)
- Support for multiplatform container imege \(amd64 and arm64\) [\#1664](https://github.com/kubeflow/training-operator/issues/1664)
- Training Operator pod failed to start on OCP 4.10.30 with error "memory limit too low" [\#1661](https://github.com/kubeflow/training-operator/issues/1661)
- After setting hostNetwork to true, mpi does not work [\#1657](https://github.com/kubeflow/training-operator/issues/1657)
- What is the purpose of /examples/pytorch/elastic/etcd.yaml [\#1655](https://github.com/kubeflow/training-operator/issues/1655)
- When will MPIJob support v2beta1 version? [\#1653](https://github.com/kubeflow/training-operator/issues/1653)
- Kubernetes HPA doesn't work with elastic PytorchJob [\#1645](https://github.com/kubeflow/training-operator/issues/1645)
- training-operator can not get podgroup status\(inqueue\) with volcano when enable gang [\#1630](https://github.com/kubeflow/training-operator/issues/1630)
- Training operator fails to create HPA for TorchElastic jobs [\#1626](https://github.com/kubeflow/training-operator/issues/1626)
- Release v1.5.0 tracking [\#1622](https://github.com/kubeflow/training-operator/issues/1622)
- upgrade client-go [\#1599](https://github.com/kubeflow/training-operator/issues/1599)
- trainning-operator may need to monitor PodGroup [\#1574](https://github.com/kubeflow/training-operator/issues/1574)
- Error: invalid memory address or nil pointer dereference [\#1553](https://github.com/kubeflow/training-operator/issues/1553)
- The pytorchJob training is slow [\#1532](https://github.com/kubeflow/training-operator/issues/1532)
- pytorch elastic scheduler error [\#1504](https://github.com/kubeflow/training-operator/issues/1504)

## [v1.4.0-rc.0](https://github.com/kubeflow/training-operator/tree/v1.4.0-rc.0) (2022-01-26)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.3.0...v1.4.0-rc.0)

**Features and improvements:**

- Display coverage % in GitHub actions list [\#1442](https://github.com/kubeflow/training-operator/issues/1442)
- Add Go test to CI [\#1436](https://github.com/kubeflow/training-operator/issues/1436)

**Fixed bugs:**

- \[bug\] Missing init container in PyTorchJob [\#1482](https://github.com/kubeflow/training-operator/issues/1482)
- Fail to install tf-operator in minikube because of the version of kubectl/kustomize [\#1381](https://github.com/kubeflow/training-operator/issues/1381)

**Closed issues:**

- Restore KUBEFLOW\_NAMESPACE options [\#1522](https://github.com/kubeflow/training-operator/issues/1522)
- Improve test coverage  [\#1497](https://github.com/kubeflow/training-operator/issues/1497)
- swagger.json missing Pytorchjob.Spec.ElasticPolicy [\#1483](https://github.com/kubeflow/training-operator/issues/1483)
- PytorchJob DDP training will stop if I delete a worker pod [\#1478](https://github.com/kubeflow/training-operator/issues/1478)
- Write down e2e failure debug process [\#1467](https://github.com/kubeflow/training-operator/issues/1467)
- How can i add the Priorityclass to the TFjob？ [\#1466](https://github.com/kubeflow/training-operator/issues/1466)
- github.com/go-logr/zapr.\(\*zapLogger\).Error [\#1444](https://github.com/kubeflow/training-operator/issues/1444)
- Podgroup is constantly created and deleted after tfjob is success or failure [\#1426](https://github.com/kubeflow/training-operator/issues/1426)
- Cut official release of 1.3.0 [\#1425](https://github.com/kubeflow/training-operator/issues/1425)
- Add "not maintained" notice to other operator repos [\#1423](https://github.com/kubeflow/training-operator/issues/1423)
- Python SDK for Kubeflow Training Operator  [\#1380](https://github.com/kubeflow/training-operator/issues/1380)

**Merged pull requests:**

- Update manifests with latest image tag [\#1527](https://github.com/kubeflow/training-operator/pull/1527) ([johnugeorge](https://github.com/johnugeorge))
- add option for mpi kubectl delivery [\#1525](https://github.com/kubeflow/training-operator/pull/1525) ([zw0610](https://github.com/zw0610))
- restore option namespace in launch arguments [\#1524](https://github.com/kubeflow/training-operator/pull/1524) ([zw0610](https://github.com/zw0610))
- remove unused scripts [\#1521](https://github.com/kubeflow/training-operator/pull/1521) ([zw0610](https://github.com/zw0610))
- remove ChanYiLin from approvers [\#1513](https://github.com/kubeflow/training-operator/pull/1513) ([ChanYiLin](https://github.com/ChanYiLin))
- add StacktraceLevel for zapr [\#1512](https://github.com/kubeflow/training-operator/pull/1512) ([qiankunli](https://github.com/qiankunli))
- add unit tests for tensorflow controller [\#1511](https://github.com/kubeflow/training-operator/pull/1511) ([zw0610](https://github.com/zw0610))
- add the example of MPIJob [\#1508](https://github.com/kubeflow/training-operator/pull/1508) ([hackerboy01](https://github.com/hackerboy01))
- Added 2022 roadmap and migrated previous roadmap from kubeflow/common [\#1500](https://github.com/kubeflow/training-operator/pull/1500) ([terrytangyuan](https://github.com/terrytangyuan))
- Fix a typo in mpi controller log [\#1495](https://github.com/kubeflow/training-operator/pull/1495) ([LuBingtan](https://github.com/LuBingtan))
- feat\(pytorch\): Add init container config to avoid DNS lookup failure [\#1493](https://github.com/kubeflow/training-operator/pull/1493) ([gaocegege](https://github.com/gaocegege))
- chore: Fix GitHub Actions script [\#1491](https://github.com/kubeflow/training-operator/pull/1491) ([tenzen-y](https://github.com/tenzen-y))
- chore: Fix missspell in tfjob [\#1490](https://github.com/kubeflow/training-operator/pull/1490) ([tenzen-y](https://github.com/tenzen-y))
- chore: Update OWNERS [\#1489](https://github.com/kubeflow/training-operator/pull/1489) ([gaocegege](https://github.com/gaocegege))
- Bump jinja2 from 2.10.1 to 2.11.3 in /py/kubeflow/tf\_operator [\#1487](https://github.com/kubeflow/training-operator/pull/1487) ([dependabot[bot]](https://github.com/apps/dependabot))
- fix comments for mpi-controller [\#1485](https://github.com/kubeflow/training-operator/pull/1485) ([hackerboy01](https://github.com/hackerboy01))
- add expectation-related functions for other resources used in mpi-controller [\#1484](https://github.com/kubeflow/training-operator/pull/1484) ([zw0610](https://github.com/zw0610))
- Add MPI job to README now that it's supported [\#1480](https://github.com/kubeflow/training-operator/pull/1480) ([terrytangyuan](https://github.com/terrytangyuan))
- add mpi doc [\#1477](https://github.com/kubeflow/training-operator/pull/1477) ([zw0610](https://github.com/zw0610))
- Set Go version of base image to 1.17 [\#1476](https://github.com/kubeflow/training-operator/pull/1476) ([tenzen-y](https://github.com/tenzen-y))
- update label for tf-controller [\#1474](https://github.com/kubeflow/training-operator/pull/1474) ([zw0610](https://github.com/zw0610))
- Add Akuity to the list of adopters [\#1473](https://github.com/kubeflow/training-operator/pull/1473) ([terrytangyuan](https://github.com/terrytangyuan))
- Add PR template with doc checklist [\#1470](https://github.com/kubeflow/training-operator/pull/1470) ([andreyvelich](https://github.com/andreyvelich))
- Add e2e failure debugging guidance [\#1469](https://github.com/kubeflow/training-operator/pull/1469) ([Jeffwan](https://github.com/Jeffwan))
- chore: Add .gitattributes to ignore Jsonnet test code for linguist [\#1463](https://github.com/kubeflow/training-operator/pull/1463) ([terrytangyuan](https://github.com/terrytangyuan))
- Migrate additional examples from xgboost-operator [\#1461](https://github.com/kubeflow/training-operator/pull/1461) ([terrytangyuan](https://github.com/terrytangyuan))
- Minor edits to README.md [\#1460](https://github.com/kubeflow/training-operator/pull/1460) ([terrytangyuan](https://github.com/terrytangyuan))
- add mpi-operator\(v1\) to the unified operator [\#1457](https://github.com/kubeflow/training-operator/pull/1457) ([hackerboy01](https://github.com/hackerboy01))
- fix tfjob status when enableDynamicWorker set true [\#1455](https://github.com/kubeflow/training-operator/pull/1455) ([zw0610](https://github.com/zw0610))
- feat\(pytorch\): Support elastic training [\#1453](https://github.com/kubeflow/training-operator/pull/1453) ([gaocegege](https://github.com/gaocegege))
- fix: generate printer columns for job crds [\#1451](https://github.com/kubeflow/training-operator/pull/1451) ([henrysecond1](https://github.com/henrysecond1))
- Fix README typo [\#1450](https://github.com/kubeflow/training-operator/pull/1450) ([davidxia](https://github.com/davidxia))
- consistent naming for better readability [\#1449](https://github.com/kubeflow/training-operator/pull/1449) ([pramodrj07](https://github.com/pramodrj07))
- Fix set scheduler error [\#1448](https://github.com/kubeflow/training-operator/pull/1448) ([qiankunli](https://github.com/qiankunli))
- Add CI to run the tests for Go [\#1440](https://github.com/kubeflow/training-operator/pull/1440) ([tenzen-y](https://github.com/tenzen-y))
- fix: Add missing retrying package that failed the import [\#1439](https://github.com/kubeflow/training-operator/pull/1439) ([terrytangyuan](https://github.com/terrytangyuan))
- Generate a single `swagger.json` file for all frameworks [\#1437](https://github.com/kubeflow/training-operator/pull/1437) ([alembiewski](https://github.com/alembiewski))
- Update links and files with the new URL [\#1434](https://github.com/kubeflow/training-operator/pull/1434) ([andreyvelich](https://github.com/andreyvelich))
- chore: update CHANGELOG.md [\#1432](https://github.com/kubeflow/training-operator/pull/1432) ([Jeffwan](https://github.com/Jeffwan))
- Add acknowledgement section in README to credit all contributors [\#1422](https://github.com/kubeflow/training-operator/pull/1422) ([terrytangyuan](https://github.com/terrytangyuan))
- Add Cisco to Adopters List [\#1421](https://github.com/kubeflow/training-operator/pull/1421) ([andreyvelich](https://github.com/andreyvelich))
- Add Python SDK for Kubeflow Training Operator [\#1420](https://github.com/kubeflow/training-operator/pull/1420) ([alembiewski](https://github.com/alembiewski))
- docs: Move myself to approvers [\#1419](https://github.com/kubeflow/training-operator/pull/1419) ([terrytangyuan](https://github.com/terrytangyuan))
- fix hyperlinks in the 'overview' section [\#1418](https://github.com/kubeflow/training-operator/pull/1418) ([pramodrj07](https://github.com/pramodrj07))
- docs: Migrate adopters of all operators to this repo [\#1417](https://github.com/kubeflow/training-operator/pull/1417) ([terrytangyuan](https://github.com/terrytangyuan))
- Feature/support pytorchjob set queue of volcano [\#1415](https://github.com/kubeflow/training-operator/pull/1415) ([qiankunli](https://github.com/qiankunli))
- Bump controller-tools to 0.6.0 and enable GenerateEmbeddedObjectMeta [\#1409](https://github.com/kubeflow/training-operator/pull/1409) ([Jeffwan](https://github.com/Jeffwan))
- Update scripts to generate sdk for all frameworks [\#1389](https://github.com/kubeflow/training-operator/pull/1389) ([Jeffwan](https://github.com/Jeffwan))

## [v1.3.0](https://github.com/kubeflow/training-operator/tree/v1.3.0) (2021-10-03)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.3.0-rc.2...v1.3.0)

**Fixed bugs:**

- Unable to specify pod template metadata for TFJob [\#1403](https://github.com/kubeflow/training-operator/issues/1403)

## [v1.3.0-rc.2](https://github.com/kubeflow/training-operator/tree/v1.3.0-rc.2) (2021-09-21)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.3.0-rc.1...v1.3.0-rc.2)

**Fixed bugs:**

- Missing Pod label for Service selector [\#1399](https://github.com/kubeflow/training-operator/issues/1399)

## [v1.3.0-rc.1](https://github.com/kubeflow/training-operator/tree/v1.3.0-rc.1) (2021-09-15)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.3.0-rc.0...v1.3.0-rc.1)

**Fixed bugs:**

- \[bug\] Reconcilation fails when upgrading common to 0.3.6 [\#1394](https://github.com/kubeflow/training-operator/issues/1394)

**Merged pull requests:**

- Update manifests with  latest image tag [\#1406](https://github.com/kubeflow/training-operator/pull/1406) ([johnugeorge](https://github.com/johnugeorge))
- 2010: fix to expose correct monitoring port  [\#1405](https://github.com/kubeflow/training-operator/pull/1405) ([deepak-muley](https://github.com/deepak-muley))
- Fix 1399: added pod matching label in service selector [\#1404](https://github.com/kubeflow/training-operator/pull/1404) ([deepak-muley](https://github.com/deepak-muley))
- fix: runPolicy validation error in the examples [\#1401](https://github.com/kubeflow/training-operator/pull/1401) ([Jeffwan](https://github.com/Jeffwan))

## [v1.3.0-rc.0](https://github.com/kubeflow/training-operator/tree/v1.3.0-rc.0) (2021-08-31)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.3.0-alpha.3...v1.3.0-rc.0)

**Merged pull requests:**

- chore: Update training-operator tag [\#1396](https://github.com/kubeflow/training-operator/pull/1396) ([Jeffwan](https://github.com/Jeffwan))
- Add simple verification jobs [\#1391](https://github.com/kubeflow/training-operator/pull/1391) ([Jeffwan](https://github.com/Jeffwan))
- fix: volcano pod group creation issue [\#1390](https://github.com/kubeflow/training-operator/pull/1390) ([Jeffwan](https://github.com/Jeffwan))
- chore: Bump kubeflow/common version to 0.3.7 [\#1388](https://github.com/kubeflow/training-operator/pull/1388) ([Jeffwan](https://github.com/Jeffwan))

## [v1.3.0-alpha.3](https://github.com/kubeflow/training-operator/tree/v1.3.0-alpha.3) (2021-08-29)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.2.1...v1.3.0-alpha.3)

**Closed issues:**

- Update guidance to install all-in-one operator in README.md  [\#1386](https://github.com/kubeflow/training-operator/issues/1386)

**Merged pull requests:**

- chore\(doc\): Update README.md [\#1387](https://github.com/kubeflow/training-operator/pull/1387) ([Jeffwan](https://github.com/Jeffwan))
- Remove tf-operator from the codebase [\#1378](https://github.com/kubeflow/training-operator/pull/1378) ([thunderboltsid](https://github.com/thunderboltsid))

## [v1.2.1](https://github.com/kubeflow/training-operator/tree/v1.2.1) (2021-08-27)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.3.0-alpha.2...v1.2.1)

## [v1.3.0-alpha.2](https://github.com/kubeflow/training-operator/tree/v1.3.0-alpha.2) (2021-08-15)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.3.0-alpha.1...v1.3.0-alpha.2)

## [v1.3.0-alpha.1](https://github.com/kubeflow/training-operator/tree/v1.3.0-alpha.1) (2021-08-13)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.2.0...v1.3.0-alpha.1)

## [v1.2.0](https://github.com/kubeflow/training-operator/tree/v1.2.0) (2021-08-03)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.1.0...v1.2.0)

## [v1.1.0](https://github.com/kubeflow/training-operator/tree/v1.1.0) (2021-03-20)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.0.1-rc.5...v1.1.0)

## [v1.0.1-rc.5](https://github.com/kubeflow/training-operator/tree/v1.0.1-rc.5) (2021-02-09)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.0.1-rc.4...v1.0.1-rc.5)

## [v1.0.1-rc.4](https://github.com/kubeflow/training-operator/tree/v1.0.1-rc.4) (2021-02-04)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.0.1-rc.3...v1.0.1-rc.4)

## [v1.0.1-rc.3](https://github.com/kubeflow/training-operator/tree/v1.0.1-rc.3) (2021-01-27)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.0.1-rc.2...v1.0.1-rc.3)

## [v1.0.1-rc.2](https://github.com/kubeflow/training-operator/tree/v1.0.1-rc.2) (2021-01-27)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.0.1-rc.1...v1.0.1-rc.2)

## [v1.0.1-rc.1](https://github.com/kubeflow/training-operator/tree/v1.0.1-rc.1) (2021-01-18)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.0.1-rc.0...v1.0.1-rc.1)

## [v1.0.1-rc.0](https://github.com/kubeflow/training-operator/tree/v1.0.1-rc.0) (2020-12-22)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v1.0.0-rc.0...v1.0.1-rc.0)

## [v1.0.0-rc.0](https://github.com/kubeflow/training-operator/tree/v1.0.0-rc.0) (2019-06-24)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v0.5.3...v1.0.0-rc.0)

## [v0.5.3](https://github.com/kubeflow/training-operator/tree/v0.5.3) (2019-06-03)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v0.5.2...v0.5.3)

## [v0.5.2](https://github.com/kubeflow/training-operator/tree/v0.5.2) (2019-05-23)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v0.5.1...v0.5.2)

## [v0.5.1](https://github.com/kubeflow/training-operator/tree/v0.5.1) (2019-05-15)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v0.5.0...v0.5.1)

## [v0.5.0](https://github.com/kubeflow/training-operator/tree/v0.5.0) (2019-03-26)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v0.4.0...v0.5.0)

## [v0.4.0](https://github.com/kubeflow/training-operator/tree/v0.4.0) (2019-02-13)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v0.4.0-rc.1...v0.4.0)

## [v0.4.0-rc.1](https://github.com/kubeflow/training-operator/tree/v0.4.0-rc.1) (2018-11-28)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v0.4.0-rc.0...v0.4.0-rc.1)

## [v0.4.0-rc.0](https://github.com/kubeflow/training-operator/tree/v0.4.0-rc.0) (2018-11-19)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v0.3.0...v0.4.0-rc.0)

## [v0.3.0](https://github.com/kubeflow/training-operator/tree/v0.3.0) (2018-09-22)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v0.2.0-rc1...v0.3.0)

## [v0.2.0-rc1](https://github.com/kubeflow/training-operator/tree/v0.2.0-rc1) (2018-06-21)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/v0.1.0...v0.2.0-rc1)

## [v0.1.0](https://github.com/kubeflow/training-operator/tree/v0.1.0) (2018-03-29)

[Full Changelog](https://github.com/kubeflow/training-operator/compare/5b1ff9c7058c2af718ed8d399aebcfd124217f8c...v0.1.0)



\* *This Changelog was automatically generated by [github_changelog_generator](https://github.com/github-changelog-generator/github-changelog-generator)*
