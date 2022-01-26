# Changelog

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
