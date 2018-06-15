[33mcommit 2eb2bcc78c3753bb3e0052ed6255dd34c8e16254[m[33m ([m[1;36mHEAD -> [m[1;32mtfjob-status[m[33m)[m
Author: Penghui Yan <yanpenghui@caicloud.io>
Date:   Tue Jun 12 14:57:13 2018 +0200

    test

[33mcommit 9bbe95cdce7e8958ae6182b2c5fbb83ca07a4e13[m[33m ([m[1;31myph/tfjob-status[m[33m)[m
Author: Penghui Yan <yanpenghui@caicloud.io>
Date:   Tue Jun 12 13:56:22 2018 +0200

    update status

[33mcommit c27036ad42516933aa467c48bc2dc5e3c9eec851[m
Author: Penghui Yan <yanpenghui@caicloud.io>
Date:   Tue Jun 5 14:33:06 2018 +0200

    add errcheck

[33mcommit c2e0a97fd3940c8e3eb1391afe20821db43ae493[m
Author: Penghui Yan <yanpenghui@caicloud.io>
Date:   Tue Jun 5 10:54:10 2018 +0200

    update curentCondition

[33mcommit 3d227d873f9b791d1efca2f056138290a6670857[m
Author: Penghui Yan <yanpenghui@caicloud.io>
Date:   Tue Jun 5 10:47:57 2018 +0200

    [v1alpha2]add distributed state management

[33mcommit 296aa5b63044e4b8e00a6238ad18515e579e18bc[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Tue Jun 5 13:01:57 2018 +0800

    pkg: Support customized port (#621)
    
    * pkg: Support customized port
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * test: Fix
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * test: Fix
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * pod: Remove hard code
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * tf: Update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit ea770be0edc576940abf293bcbe4dc98c2b219dd[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Tue Jun 5 11:05:56 2018 +0800

    pkg: Add event send statements (#623)
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit f9430f1028cb948f8f6fc49a140a26e3fb8923dc[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Mon Jun 4 21:49:54 2018 +0800

    controller_service: Headless service (#576)
    
    * service: Use headless service
    
    * controller: Use headless service
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 93843edae6b2c00cfa52a83df89e2592b1832662[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Mon Jun 4 16:23:54 2018 +0800

    service_control: Set owner ref for service and add test cases (#617)
    
    * service_control: Set owner ref for service
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * service_control: Add err check
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 0be3b81ae98e1b09f062594e241501337a99f268[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Mon Jun 4 15:44:54 2018 +0800

    API: Fix (#618)
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 5dc96cecc495b76ba56ae945a631044247090b1c[m
Author: Ankush Agarwal <agwl@google.com>
Date:   Sun Jun 3 21:03:54 2018 -0700

    Add apiVersion parameter to simple_tfjob component (#619)

[33mcommit 2cfe1bad4720379e10b32a0af4cc4d72d58d9b16[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Fri Jun 1 03:00:53 2018 -0400

    [dashboard] Upgrade to v1alpha2 (#613)
    
    * refactor dashboard to v1alpha2
    
    * Migrate tfjob dashboard to v1alpha2

[33mcommit 924e927952da5ec7ef0b3d6c1700155c85332410[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Fri Jun 1 14:57:58 2018 +0800

    test: Add test cases for service ref manager and control interface (#615)
    
    * vendor: Update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * test: Add
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * test: Add ref manager test
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 485d430f864377423d9ce836dc902816f10f2153[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Fri Jun 1 12:57:56 2018 +0800

    controller: Refactor and add test cases for helper (#614)
    
    * *: Refactor
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * test: Fix
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 7783f7891621ecead82d3c403db14d205f2a3298[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Thu May 31 18:24:54 2018 +0800

    controller: Improve coding styles (#612)
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit cb18925db330918a65f9119b4060417c5ada5992[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Thu May 31 14:59:56 2018 +0800

    api: OpenAPI support (#577)
    
    * hack: Add openapi-gen
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * api: Generate openapi model
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * linter_config: Add
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * openapi: Add k8s.io
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * .travis: Ignore openapi
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * dep: Update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit a9294ce4ae5604d44b62d67dbc9d2287891c7767[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Thu May 31 14:00:55 2018 +0800

    Informer: Use unstructured (#610)
    
    * vendor: Update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * gopkg: Update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * controller: Use unstructured informer
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit ad6745b820cf7ce643da5ff9a71f1b3a1743c51a[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Thu May 31 13:11:56 2018 +0800

    service: Refactor to the slice structure (#603)
    
    * service: Refactor to the slice structure
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * controller.go: Remove useless code
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * controller: Fix test
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * service: Fix comments
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit e8e152e44d73c938c680d4863ab943e6fa39ccc6[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Fri May 25 21:52:33 2018 +0800

    crd: Add validation using OpenAPI 3.0 (#605)
    
    * crd: Add validation using OpenAPI 3.0
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * crd-v1alpha2: Add comment
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 98b777eba0b9e91d55be2fd917f25a198cdddeef[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Fri May 25 09:19:54 2018 +0200

    TFJob client should not block forever trying to get the namespace object (#607)
    
    * TFJob client should not block forever trying to get the namespace object.
    
    * TFJob wait should run the request asyncronously so we don't end up blocking
      forever.
    
    Fix #606
    
    * Fix lint.

[33mcommit 6a5ee893ce493cf25b2ba1f7fef746fa5895d187[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Wed May 23 09:55:25 2018 +0800

    dist_mnist: Add unused_argv (#604)
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 77375baf7b71d9caea88ab6457e3d5e4484c1b5d[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue May 22 05:52:54 2018 +0200

    Add tf-operator.v2 to release.py so that we build a Docker image containing (#601)
    
    the v1alpha2 controller.
    
    Update the dockerfile to install the v1alpha2 operator.
    
    Fix #600

[33mcommit 8cf2831278b854f8b142c6075c16cad6941e7b64[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue May 22 04:29:55 2018 +0200

    Delete the old releaser code which is no longer used. (#602)
    
    * We now build and release the images using our Argo workflows which
      calls py.release.
    
    * This code was used for our old continuous builds and helm.

[33mcommit 9be6c5d57f0c0ec4d8c3557e962e60843fabcbc4[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Sun May 20 23:15:49 2018 +0800

    controller: Remove dup code and use k8s.io/kubernetes/controller (#594)
    
    * dep: Update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * vendor: Update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * *: Use k8s.io/kubernetes/controller
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * test: Fix
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * test: Fix
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * test: Fix
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 8876ebb59b6ce26261921e412911519144f42be2[m
Author: Chaolei Li <chaoleili2@gmail.com>
Date:   Thu May 17 21:33:50 2018 +0800

    Add a new command-line argument for release.py (#595)
    
    * Add a new command-line argument for release
    
    Add --push_with_docker argument for release.py. When using
    --push_with_docker, the release will use docker to push
    image.
    
    * Add a new command-line argument for release
    
    Add --push_with_docker argument for release.py. When using
    --push_with_docker, the release will use docker to push
    the image.
    
    * Update docker push method
    
    According image's registry automaticlly choose push method.
    The gcr.io will using gcloud, others will use docker to push
    image.

[33mcommit e984dd2328c871b1831a3ca2fec25906001a8c66[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Mon May 14 22:29:25 2018 +0800

    docs: Add quick start for v1alpah2 (#584)
    
    * docs: Add
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * docs: Update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * quick-start: Update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit e13b4c3c31b5460dd1e91bd66c44ed9b03ce6b51[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Mon May 14 13:22:24 2018 +0800

    test: Fix data race problem (#593)
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 2918f08c0d14e26a37ae81cd9679480ec7254cb1[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Fri May 11 19:48:55 2018 +0800

    .travis.yml: Fix cmd errors (#592)
    
    * .travis.yml: Fix cmd errors
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * dashboard: Fix
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * backend: Fix
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * *: Fix
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * api: Fix err variable name
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 2cdf4d28a37796e6b91b64affab87336431baad2[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Fri May 11 15:30:56 2018 +0800

    mnist: Add correponding yaml config (#583)
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit a2d7f80368495a6f067edb7555a62e491283276d[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Fri May 11 15:05:55 2018 +0800

    controller_status: Remove pending pods from active pods (#578)
    
    * controller_status: Fix #484
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * controller: Fix test cases
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit f0c21f8917d7e21bf7f40d83ddbc1a0e3eb19475[m
Author: Jack(Guoliang) Wang <iamwgliang@gmail.com>
Date:   Fri May 11 14:01:46 2018 +0800

    Fix the gometalinter support (#587)

[33mcommit 7b40b9ac32848dea4d864b76847b4f94f7727ff7[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Fri May 11 11:23:55 2018 +0800

    pkg: Add update (#582)
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 0ab60603c1fef5a9b3e4c30f8c4227f50ca7c4dd[m
Author: Jack(Guoliang) Wang <iamwgliang@gmail.com>
Date:   Thu May 10 18:55:53 2018 +0800

    Format go code and fix spelling errors (#585)

[33mcommit e71d96b95fa669e9bfdf56a05c12f71dccae298b[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Thu May 10 17:30:54 2018 +0800

    .pylinrc: Add dist_mnist (#581)
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit e38058023f722f8bffb87163676e47563d252f0b[m
Author: Penghao Cen <scorpiocph@gmail.com>
Date:   Thu May 10 10:46:51 2018 +0800

    Add dist mnist model for e2e test (#549)

[33mcommit 0c8ffe81c6db6373fd8ae3d7b614358b700b276e[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Thu May 10 10:09:54 2018 +0800

    .travis.yml: Add failure notification in GitHub (#579)
    
    * .travis.yml: Add failure notification in GitHub
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * .travis.yml: Fix space
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 556e2fc544d46f06a1f8212008fbe60248474f7f[m
Author: Jack <j5111261112@gmail.com>
Date:   Tue May 8 09:55:51 2018 +0800

    replace glide with dep in the developer guide (#572)

[33mcommit e69e88d1cb3e607ce5ea3e5af331e1dbc59403fd[m
Author: Penghui Yan <1846523352@qq.com>
Date:   Mon May 7 18:18:51 2018 +0800

    [v1alpha2]fix bug int to string for index (#571)

[33mcommit 0de755c70abc27a208a79571a2634423e42e2e40[m
Author: Zachary Zhao <zacharyzhao@yahoo.com>
Date:   Mon May 7 16:08:51 2018 +0800

    Fix missing string for logging placeholder (#570)

[33mcommit 2a22ad490d2bed946e812c9c8b166166f6ea82e4[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Sun May 6 00:26:51 2018 +0800

    chart: Remove (#566)
    
    * chart: Remove
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * py: Fix lint issues
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * release_test: Revert
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit f317193e976606e43d16a0457659b59dc0123229[m
Author: Jack <j5111261112@gmail.com>
Date:   Sat May 5 13:31:51 2018 +0800

    add OwnerReferences to pdb (#565)
    
    * add OwnerReferences to pdb
    
    * refactor the syncPdb code
    
    * fix the bug of access pdb name.
    
    * add fake ObjectMeta uid in the testcase of TestPDBForGangScheduling
    
    * fix error when no pdb should be created
    
    * remove the fake uid in the testcase

[33mcommit bab9d9ecfd0a2a6788a37cba4797c0155038f3a0[m
Author: Ankush Agarwal <agwl@google.com>
Date:   Thu May 3 19:18:50 2018 -0700

    Update py_lint and py_test (#569)
    
    * Update py_lint and py_test
    
    https://github.com/kubeflow/testing/pull/111 refactored these two tests
    into two separate files
    
    /cc @gaocegege
    
    * Ignore jupyterhub_spawner.py

[33mcommit 19d611cf4a94b1cd4fd86125ec36aeea8220595d[m
Author: Ankush Agarwal <agwl@google.com>
Date:   Wed May 2 07:28:17 2018 -0700

    Update test worker image to kubeflow-ci (#568)
    
    * Update test worker image to kubeflow-ci
    
    Related https://github.com/kubeflow/testing/issues/90
    
    * Replace all occurences of mlkube-testing

[33mcommit 4a23ad0adc00866ad78db09d6aa1a27a79f8ad1c[m
Author: Ce Gao <5100735+gaocegege@users.noreply.github.com>
Date:   Thu Apr 26 13:51:02 2018 +0800

    vendor: Use dep instead of glide and prune it (#557)
    
    * vendor: Update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * pkg: Regenerate
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * update-codegen: Update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * vendor: Replace glide
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 07fd2d6344384af8c4a56b038d0bc975c323d617[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Wed Apr 25 15:50:01 2018 +0800

    controller: Refactor controller_pod (#548)
    
    * controller_pod: refactor
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * controller: Add test case
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * WIP
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * controller_pod: Separate creation and status update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * controller: Add continue
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * controller: Remove old code
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * controller: Remove log
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * controller: Update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * controller_status: Add copyright header
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit a3dbdfef5efe4451b1ae968bd40234fe6cd7fdbd[m
Author: Neil Tenenholtz <ntenenz@users.noreply.github.com>
Date:   Tue Apr 24 23:25:02 2018 -0400

    Correct typos in README (#559)

[33mcommit 60d25c4335583260027e6bebc5e28f0305a56714[m
Author: leiiwang <u2takey@gmail.com>
Date:   Tue Apr 24 11:26:45 2018 +0800

    set completion time on success (#554)

[33mcommit 33bc77421d0c823f9824f4d3bade63d07eb87b98[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Mon Apr 23 10:32:00 2018 +0800

    README: Add tf-operator v1alpha2 design doc (#553)
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 42ff961a525eab42d9495b537d04a8fb61eec9d8[m
Author: Trevor McKay <tmckay@redhat.com>
Date:   Fri Apr 20 21:49:00 2018 -0400

    Use a CentOS 7 base image for the tf-operator image (#469)
    
    * Use a CentOS base image for the tf-operator image
    
    * Remove comment about golang-alpine (doesn't apply)
    
    * Install golang from an rpm rather than a url
    
    * Resolve pylint errors in release and release_test

[33mcommit c4ad78964cdf0050e9623d111d31aaf142d3d802[m
Author: 0olwzo0 <wenzhel@google.com>
Date:   Fri Apr 20 16:57:00 2018 -0700

    Only identify specific exit codes as retryable error (#518)
    
    * Should return ReplicaStateFailed if container exit code is not 0
    
    * Update the criteria for retryable errors.
    
    * Reformat
    
    * Reformat
    
    * Reformat
    
    * Fix lint error.
    
    * Handle the exit code more explicitly.
    
    * Reformat.
    
    * Create a util func for IsRetryableExitCode.

[33mcommit 9e07f6df0c6431b225a60f91f65997f658d2b7f7[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Fri Apr 20 14:57:59 2018 +0800

    v1alpha2: Add implementation (#526)
    
    * *: Add implementation for v1alpha2
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * linter: Fix
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * options: Remove default config for kubeconfig
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * service: Add const
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * server.go: Remove new lines
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * *: Fix import styles
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * controller.go: Add comment for handler
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * controller_pod: Fix name
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * options: Remove kubeconfig
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * *: Update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * controller_pod: Add TODO
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit a2538f851ab3c4b0ffeae80d8c4f2bffa1ab00ea[m
Author: Ankush Agarwal <agwl@google.com>
Date:   Thu Apr 19 19:20:00 2018 -0700

    Replace kubeflow-images-staging with kubeflow-images-public (#546)
    
    Related to kubeflow/kubeflow#534

[33mcommit ff2958ad37cc5499322dcd0b017b565291ca8ea3[m
Author: leiiwang <u2takey@gmail.com>
Date:   Wed Apr 18 17:35:00 2018 +0800

    copy labels and anotations to pod from tfjob (#542)

[33mcommit 846fd38332249ddbb348e28eb9056a0731ce13f4[m
Author: Jack <j5111261112@gmail.com>
Date:   Tue Apr 17 10:16:59 2018 +0800

    fix the bug of keeping creating new pdb (#539)
    
    * fix the bug of keeping creating new pdb
    
    * fix the error in Travis CI

[33mcommit fd95c5e8b5e6032b86992e806d49aeb5f853df7a[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Thu Apr 12 12:33:57 2018 +0800

    OWNERS: Add @ddysher and @willb as reviewers (#529)
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 6e706d6071b28f36e8e96d3d3e5f5ebaba574631[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Thu Apr 12 10:41:55 2018 +0800

    signals: Add (#531)
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit ecd792c089d82774ec74c78037e9c5e4538d63f9[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Wed Apr 11 21:46:54 2018 +0800

    developer_guide: Add instructions for v1alpha2 (#528)
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 792d399e6538d92f80258e58e418cd92759f7f3f[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Tue Apr 10 21:15:10 2018 +0800

    v1alpha2: Add API and codegen (#523)
    
    * types.go: Add APIs
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * apis: Related code
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * update-codegen.sh: Update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * codegen: Update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * linter: Update
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * types.go: Fix typo
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * test: Remove Pformat
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 0704e24dbd3b5ea720e1e23e854b516905396c3d[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Mon Apr 2 16:43:15 2018 -0700

    Reenable cluster teardown. (#520)

[33mcommit a83ccdc54e4cda8edc01798ce8e461a1835ea226[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Mon Apr 2 16:23:15 2018 -0700

    Create a script to release the TFJob operator image (#515)
    
    * Create instructions for doing a release of TFJob operator docker image
    
    * Add a script to submit the Argo workflow.
    
    * * Add instructions for how to do a release.
    * Update the submit_release_job.sh so it can run from anywhere in the tree.
    
    * * Get the root directory based on the script.

[33mcommit c5c8c26c77d5d86b7e58ad519fdb8a3de72ea630[m
Author: Hitoshi Mitake <mitake.hitoshi@gmail.com>
Date:   Fri Mar 30 16:20:03 2018 -0700

    update OWNERS (#516)
    
    * Add mitake as a reviewer
    * Remove zjj2wry (based on a request from jlewi)

[33mcommit 85dd1a328d35dac5224cf559f34fbd7fd2cfe862[m
Author: Jose Aguirre <jose5918@gmail.com>
Date:   Fri Mar 30 10:05:02 2018 -0700

    Adds gcloudignore (#510)
    
    Motivation for this is to not upload the entire vendor directory (or other potentially unecessary files) when building a container

[33mcommit be41326985e15663aa04f92ddd2b34e5e34ef178[m
Author: Jose Aguirre <jose5918@gmail.com>
Date:   Thu Mar 29 20:14:03 2018 -0700

    Fix output on test failure (#511)

[33mcommit c69acf89545abdc9b6308f4a956f40f704a5ef23[m
Author: Hitoshi Mitake <mitake.hitoshi@gmail.com>
Date:   Thu Mar 29 19:56:03 2018 -0700

    Add a new command for generating example TFjobs (#509)
    
    This command adds a new command genjob to hack/ directory. The command
    generates TFJobs for testing purpose. Currently the generated jobs are
    following the definitions of examples/tf_job.yaml and
    examples/tf_job_gpu.yaml.
    
    Example usage:
    ```
    $ ./genjob --kube-config-path ~/.kube/config --nr-tfjobs 30 --use-gpu --scheduler-name tfjob
    ```

[33mcommit a7511ff284621e4d88dba2354b048d57965a4011[m[33m ([m[1;33mtag: v0.1.0[m[33m, [m[1;31morigin/v0.1-branch[m[33m)[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Thu Mar 29 04:58:03 2018 -0700

    Don't leave pods running when a job completes. (#512)
    
    * Don't leave pods running when a job completes.
    
    * We originally did this to preserve the logs.
    * But this ends up leaving pods running consuming resources.
    * The fix is straightforward
      * Transition to the cleanup phase before transitioning to the done phase.
    
    Fix #128
    
    * Don't teardown the cluster.
    
    * Don't set phase to cleanup when job is running.
    
    * Should only call get status if we are in creating or running phase.
    
    * Update the E2E test
    
    * Check that pod/service event creations are recorded
    * Check that pods are deleted when job ends.
    
    * Fix lint.

[33mcommit 41a20d4296694f636fd8e203539b25aef2809911[m
Author: Rohit Agarwal <mindprince@gmail.com>
Date:   Thu Mar 29 15:43:03 2018 +0530

    Fix outdated information about GPUs in README (#513)

[33mcommit 53cf0900542ace668d050c317c9333642de17cd2[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Wed Mar 28 17:14:02 2018 -0700

    Add proxying to front-end development server. (#442)
    
    * dashboard: dev guide
    
    * fix dashboard + proxy issue
    
    * dashboard: move to hash router
    
    * simplify development workflow

[33mcommit 6214e5607a60d555c37e5d28246dcbaae0815942[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Mon Mar 26 12:39:03 2018 -0700

    Support testing on minikube. (#485)
    
    * Support testing on minikube.
    
    * When using minikube we don't want call configure_kubectl which calls
      gcloud container clusters get-credentials.
    
    * Instead we just need to call load_kube_config to load the kube config file.
    
    * Related to kubeflow/testing#6
    
    * Add a TODO.
    
    * Fix lint issue.

[33mcommit b72f47e4c7ba2c6568f3ea2d6a50e05e5ff6ab2b[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sun Mar 25 18:11:00 2018 -0700

    Fix bug with jobs not being marked as completed. (#501)
    
    * Fix bug with jobs not being marked as completed.
    
    * A bug was introduced with getting the replica status in #344 which
    switched to creating pods directly.
    
    * Our presubmits/postsubmits were failing but this went unnoticed because
    the git status check was improperly reported as succeeded.
    
    * The bug is because we try to get the pod status by name but the name
    doesn't include the random salt in the pod name.
    
    * The code in question is a legacy of when we were using job controllers and
    we first got the status of the job controller. We incorrectly changed that
    code to get the pod. The correct thing is to just list pods by label; we
    already do that in the code below so we just need to delete some code.
    
    * Don't create any resources if the DeletionTimestamp is set.
      Creating resources at this point would end blocking deletion of the object
      because the controller would create resources while we are trying to delete
      them.
    
    * Use logrus in controller.go, trainer.go, and replicas.go to log
      with fields providing information about the job and repliac.
      This makes it easy to filter logs for a particular job.
    
    * Use logrus to log the name of the job in a field.
    
    * Checking the deletiontime stamp doesn't appear to be sufficient.
    
    Use the Phase to determine whether we should create resources.
    
    * Run gofmt.
    
    * * Reset the rate limiter after every successful sync.
    * Otherwise the ratelimiter will end up delaying processing subsequent
      events which isn't what we want.
    
    * Run goimports to fix lint issues.
    
    * * Reconcile needs to update the TFJob stored in TrainingJob. This ensures
      TrainingJob has an up to date representation of the job.
    
    * Otherwise changes made to the spec won't be available to TrainingJob. For
      example, if the job is deleted by the user, the deletion timestamp will
      be set. But if we don't update the TFJob stored in TrainingJob this
      change won't be propogated.
    
    * * TrainingJob.update should log the value of the job not the pointer.
    
    * Add more comments to the code.

[33mcommit eec56b56e375f6dfc0f1c850988ed960e8243438[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Fri Mar 23 14:54:56 2018 +0800

    pkg: Fix the code changed in #486 (#497)
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit d944f7508b20bebc0d414c3385d83b4e2deb807a[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Fri Mar 23 14:42:29 2018 +0800

    release: Fix style (#498)
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit c7e39402e8daf1118bb5da878536fbd42f43c4a2[m
Author: Ayush Kumar <ayushkumar@protonmail.com>
Date:   Fri Mar 23 10:22:02 2018 +0530

    fixed some golint warning (#486)
    
    * fixed some golint warning
    
    * fixed some golint warning
    
    * updated linter_config.json

[33mcommit 771c4f9bb32d39c2eb4d9b7769ea068775f3ff93[m
Author: rc-zhang <rczcmu@gmail.com>
Date:   Wed Mar 21 04:12:03 2018 -0400

    Use headless services for Training jobs (#471)

[33mcommit 77bcdc48ba688b5d3bffd9070ee96893e26c8bb8[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Wed Mar 21 02:02:02 2018 -0400

    fix field selectors in controller (#465)

[33mcommit 181cb05d716201730e37942a4461270a84f696d3[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Tue Mar 20 13:06:01 2018 -0400

    Fix dashboard + proxy incompatibility (#441)
    
    * dashboard: dev guide
    
    * fix dashboard + proxy issue
    
    * dashboard: move to hash router
    
    * [dashboard] Add comments about routes and proxying

[33mcommit ebb0053e965cf131bbb45f4ea79ad74791fc5719[m
Author: Hitoshi Mitake <mitake.hitoshi@gmail.com>
Date:   Tue Mar 20 12:06:00 2018 +0900

    Create PDB of TFReplicaSet for gang scheduling by kube-arbitrator (#452)
    
    * Create PDB of TFReplicaSet for gang scheduling by kube-arbitrator
    
    kube-arbitrator (technically its component kube-batchd) requires PDB
    for its gang scheduling feature. The feature is useful for creating
    every pod of TFJob at the same time. This commit lets tf-operator
    create the PDB for the purpose.
    
    * Add a new UT TestPDBForGangScheduling for checking PDB creation

[33mcommit a36b350e04f92c8e67013c3c9ff5a0f0ee2fecd9[m
Author: rc-zhang <rczcmu@gmail.com>
Date:   Mon Mar 19 05:58:59 2018 -0400

    add LabelsByIndex method to eliminate code duplication (#474)

[33mcommit 3a279aaff42d577ac766d351c2f41f380c4ddf4c[m
Author: Lun-Kai Hsu <lunkai@google.com>
Date:   Thu Mar 15 17:59:24 2018 -0700

    Run ks upgrade (#464)
    
    * run ks upgrade
    
    * ks upgrade

[33mcommit d17792660f76777f9fd0b1d48a49450462ded9da[m
Author: Lun-Kai Hsu <lunkai@google.com>
Date:   Wed Mar 14 20:24:22 2018 -0700

    fix owners file id (#462)

[33mcommit ee2aa9b6c39344de8930802a0a852fef61485792[m
Author: Lun-Kai Hsu <lunkai@google.com>
Date:   Wed Mar 14 18:34:23 2018 -0700

    Change test cluster to kubeflow-ci (#459)
    
    * Change test cluster to kubeflow-ci
    
    * also change bucket

[33mcommit 392f5e2e412438a624721268218171c7c42c4ac3[m
Author: Penghao Cen <cenph@caicloud.io>
Date:   Tue Mar 13 21:02:09 2018 +0800

    Remove deprecated package retryutil (#460)

[33mcommit 7c627696376f0bdf6a9acc2f45ce0a6e777004e5[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Tue Mar 13 10:41:08 2018 +0800

    travis: Ignore generated code (#453)
    
    * travis: Remove generated code
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * travis: Match all files in client
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit d35eb6af3e00abe128318efadc8e2ffaf06fbbaf[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Mon Mar 12 18:05:11 2018 +0800

    *: Remove APIExtension clientset (#454)
    
    * *: Remove APIExtension clientset
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * k8sutil: Remove apiextension client
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 423b0d004bf1c877b5a587ecff8920944f036144[m
Author: Penghao Cen <cenph@caicloud.io>
Date:   Fri Mar 9 10:46:10 2018 +0800

    Create resources(services/jobs) only once (#418)

[33mcommit 83c1ab202871e6b477a862a8401c3a02a09a0c95[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Thu Mar 8 14:19:58 2018 -0500

    Add OWNERS file for dashboard (#446)
    
    * add OWNERS file for dashboard
    
    * upate OWNERS

[33mcommit c0a70bd70d0c973f217e7ee7dbff39cf60a11312[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Thu Mar 8 12:49:56 2018 -0500

    make local release xplatform (#445)

[33mcommit d3a615dd627cc0ce63a594f82f629aebf4f08c3f[m
Author: Jiayu Liu <Jimexist@users.noreply.github.com>
Date:   Wed Mar 7 22:39:53 2018 +0800

    change kubeflow.io to kubeflow.org (#440)

[33mcommit a4b8031f12b5badcf035d595f89fc8650da6d69f[m
Author: Hitoshi Mitake <mitake.hitoshi@gmail.com>
Date:   Wed Mar 7 15:27:15 2018 +0900

    Add a field SchedulerName to TFJob for specifying a scheduler (#408)
    
    This commit adds a new field SchedulerName to the definition of TFJob.
    The purpose of the field is specifying the scheduler name of the pods
    created by tf-operator and let the scheduler (which wouldn't be the
    default scheduler) handle them. It would be convenient for letting
    kube-batchd (a component of kube-arbitrator) handle the pods.

[33mcommit 997c583d18c1f2f795a9fbfa62f4a23617943dea[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Tue Mar 6 17:10:52 2018 +0800

    *: Remove type ContainerName (#432)
    
    * *: Remove type ContainerName
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * replicas_test: Fix the error
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * rebase: Fix
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit def0da0281429a156a35c89a427e8cb78127629f[m
Author: Penghao Cen <cenph@caicloud.io>
Date:   Tue Mar 6 16:35:11 2018 +0800

    Remove unreachable code (#434)
    
    As we have checked j.job.Status.Phase here before calling setup(), these codes can never be reached.

[33mcommit 6706903c1b45e775ddb310fe00fce6e07942ddb4[m
Author: Penghao Cen <cenph@caicloud.io>
Date:   Tue Mar 6 01:55:12 2018 +0800

    Create pod instead of job (#344)
    
    This PR is a part of #325:
    
    rename jobName() to genName()
    create Pod instead of Job
    
    TODOs (in another PR):
    
    use controller.PodControlInterface and CreatePodsWithControllerRef to create Pod
    Listen Pod CRUD and update TFJob status which descried in #314

[33mcommit 43677ccaa3d0da5b05e5b66ae730e6e6eac0fd25[m
Author: XsWack <xushiwei5@huawei.com>
Date:   Fri Mar 2 13:09:11 2018 +0800

    add boilerplate header for go file (#431)
    
    * add boilerplate Copyright for generated file
    
    * update update-codegen.sh

[33mcommit d7131a9010227ff3faf5baa15b2926836e4e0246[m
Author: Hitoshi Mitake <mitake.hitoshi@gmail.com>
Date:   Fri Mar 2 10:43:24 2018 +0900

    format the python files with yapf (#429)

[33mcommit 092827191bd89db0d77b31782f7cb7895bf3b2f9[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Thu Mar 1 16:29:42 2018 +0800

    fix(clientset): Fix code which is changed manually (#428)
    
    * clientset: Fix code which is changed manually
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>
    
    * pkg: Fix go import order
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 5d0d12688e08f80b41bb5eeac2da5418b73bfbbe[m
Author: Ankush Agarwal <agwl@google.com>
Date:   Wed Feb 28 19:53:34 2018 -0800

    Use logrus for structured logging (#416)
    
    * Move from glog to sirupsen/logrus for logging
    * Add a new flag json_log_format
    * Refactor all log statements to use logrus
    * Use Info level for log level V(1) and below
    * Use Debug level for log level V(2) and above
    * Tested locally
    
    Addresses https://github.com/kubeflow/tf-operator/issues/24
    
    Sample logs
    ```
    {"filename":"app/server.go:54","level":"info","msg":"EnvKubeflowNamespace not set, use default namespace","time":"2018-02-27T18:25:18-08:00"}
    {"filename":"app/server.go:59","level":"info","msg":"[Version: 0.3.0+git Git SHA: Not provided. Go Version: go1.9.3 Go OS/Arch: darwin/amd64]","time":"2018-02-27T18:25:18-08:00"}
    {"filename":"app/server.go:145","level":"info","msg":"No controller_config_file provided; using empty config.","time":"2018-02-27T18:25:18-08:00"}
    {"filename":"controller/controller.go:110","level":"info","msg":"Setting up event handlers","time":"2018-02-27T18:25:18-08:00"}
    ```

[33mcommit 2459f1c8da3df04ddfd56b04ed3069d088ba6ce5[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Feb 28 18:51:26 2018 -0800

    Delete Dockerfile to build a docker image to use for proow. (#425)
    
    * All repositories use a common image now.
    * We no longer need this image.
    
    Related to:
    
    kubeflow/kubeflow#276

[33mcommit fd4b8495f0b894ecf1b8322873ed50075d7a76b0[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Feb 28 10:28:03 2018 -0800

    Fix setup_cluster. (#421)
    
    * We need to stop pinning GKE version to 1.8.5 because that is no longer
      a valid version.
    
    * We should no longer need to pin because 1.8 is now the default.
    
    * Fix some lint issues that seem to have crept in.
    
    Fix #240

[33mcommit 1f26ef8cf9680fc500e71602d396971d79188adc[m
Author: Penghao Cen <cenph@caicloud.io>
Date:   Wed Feb 28 21:36:17 2018 +0800

    Add ScorpioCPH as approver/reviewer (#419)

[33mcommit cf97cd5f18e8b3facddbe8e33df82ecec2a2d955[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Tue Feb 27 21:22:25 2018 -0600

    dashboard: dev guide (#417)
    
    Close #411
    
    Add a developer guide for the dashboard.
    
    More doc is ultimately needed, but this is the most pressing part to unblock people from contributing.

[33mcommit 0759f7ae5142ed2e78a6971e9703fdc86b7307cd[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Wed Feb 28 04:38:56 2018 +0800

    Remove TensorBoard related code in operator (#391)
    
    Ref #347
    
    Blocked until CI is running again.
    
    PS: Dashboard code is not changed.
    
    Signed-off-by: Ce Gao ce.gao@outlook.com

[33mcommit a814670978c8d4ef78ffc3e99b370ee2d02d980c[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Feb 27 06:31:22 2018 -0800

    Create an initial OWNERS file. (#414)
    
    * This will hopefully help us spread the review, approve workload around.
    * Related to
    kubeflow/community#10

[33mcommit 0094aaa7ef501c0bc8b332edbe4a13781fe65a88[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Mon Feb 26 21:40:42 2018 -0800

    Docs should refer to Kubeflow user guide for deploying the TFJob operator. (#412)
    
    * We don't have the resources to support and maintain ksonnet and helm packages.
    * We want to focus on just using ksonnet to deploy Kubeflow.

[33mcommit c54cda92034d2cb8c2085d820a653f4065d0ed16[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Mon Feb 26 20:00:51 2018 -0800

    Support using our E2E workflow to build a Docker image for releases. (#403)
    
    Update our E2E test to use ksonnet not helm to deploy
    
    As part of this we have a slightl loss in test coverage because our
    helm test provides some verification that our python tests don't.
    Create a releasing environment for workflows that has parameters set
    as needed to build using our release cluster.
    
    Related to
    #400 Use Argo for releases
    #373 Improve our test harness

[33mcommit ab1cb4ded120ba98b33cdc290a3b5ac866ac438e[m
Author: Ankush Agarwal <agwl@google.com>
Date:   Mon Feb 26 14:55:31 2018 -0800

    Run glide update to update glide.lock (#410)
    
    I'm planning to add a new dependency - github.com/onrik/logrus/filename. To add that I need to run glide update, which updates all the dependencies in this repository. So I want to check changes unrelated to github.com/onrik/logrus/filename in a separate PR.

[33mcommit 2bcd6cebd0764fa010b2e87d776e98e57eca80ab[m
Author: Ankush Agarwal <agwl@google.com>
Date:   Mon Feb 26 12:51:20 2018 -0800

    Fix typo in Makefile (#409)

[33mcommit 32d9b17f24596bb160047f021049934ac7d1175a[m
Author: Jiayu Liu <Jimexist@users.noreply.github.com>
Date:   Tue Feb 27 04:46:41 2018 +0800

    use yapf to format python code (#401)
    
    to enforce more consistent code style.
    
    python code changes are all automated by the tool.
    
    I think yapf is better because it is more aggressive with a reasonable default. Ideally I've imagined something opinionated such as https://prettier.io/, and having such a tool means that the code format can only be in one shape. YMMV.

[33mcommit 08071b30a59a12235b1f5f9bbe58d41c047ea61c[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sat Feb 24 10:08:23 2018 -0800

    Fix lint issues with python3 and a bug in lint script (#405)
    
    Our worker image has been upgraded and its now using python3 to run the lint checks.
    There are some lint failures that only started appearing when we went to python3
    There is a bug in py_checks that only shows up if there are lint failures
    
    Fix #404

[33mcommit 32cae5d3757f561f470c724fcadbf978707ec259[m
Author: Jiayu Liu <Jimexist@users.noreply.github.com>
Date:   Sun Feb 25 00:30:49 2018 +0800

    add go 1.10 support in travis (#402)
    
    * adding go 1.10 support in travis
    
    * use string not float

[33mcommit a1ce945e3c3f2f1feb64280c91edaed3e5bd8cda[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Thu Feb 22 19:58:08 2018 -0800

    Fix bug with jobs not working if you recreate a job with same name as previous job (#399)
    
    This appears to be a bug with the informer forgetting about active jobs.
    
    I think the root cause is that delete events are processed directly
    as opposed to using the work queue. This causes race conditions because
    the work queue is what ensures only on thread is processing a given job
    at a time.
    
    Since we no longer explicitly handle delete events we rely on garbage
    collection and owner references to delete the pods.
    
    We add testing to our test runner to verify pods are garbage collected.
    
    We update test_runner.py to try recreating jobs with the same name.
    In general this doesn't reproduce the original bug.
    
    Related to #322 (to need to push a new image to fix).
    Fix #310
    
    Related to #373

[33mcommit 87c4bba9bb1a0a900577a43e8a22058f25a6e74e[m
Author: Adhita Selvaraj <adhita94@gmail.com>
Date:   Thu Feb 22 15:45:44 2018 -0500

    Addressing go vet errors (#397)
    
    Addressing #395.
    
    There are some errors in our code but it won't caught by the compilers
    
    $ go vet $(go list ./... | grep -v /vendor/)
    Output:
    
    cmd/tf-operator/app/server.go:129: unreachable code
    exit status 1
    dashboard/backend/handler/api_handler.go:36: struct field tfJobs has json tag but is not exported
    dashboard/backend/handler/api_handler.go:41: struct field namespaces has json tag but is not exported
    exit status 1

[33mcommit 398e16c724e7df25aa40147995444c53427c3ca5[m
Author: Ayush Kumar <ayushkumar@protonmail.com>
Date:   Fri Feb 16 20:25:07 2018 +0530

    [enhancement] Rename cmd/tf_operator -> cmd/tf-operator (#393)
    
    * Fixed-363: Rename cmd/tf_operator -> cmd/tf-operator
    
    * Update server.go
    
    * Update main.go
    
    * update dockerfile

[33mcommit 37f55331210f96ec3c14f0430478f5d90b3c03fc[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Thu Feb 15 12:00:55 2018 -0800

    Add a prow_config.yaml file to configure our prow jobs. (#388)
    
    See kubeflow/testing#12
    
    Our prow jobs will trigger a script that will process prow_config.yaml
    and figure out which workflows to run.
    
    This will allow us to easily run multiple workflows as part of our E2E tests.
    
    Fix one source of flakiness in TFJob test
    
    If we get the status right after creating the TFJob the status might not be set yet
    This was causing wait_for_tfjob to exit with KeyError exception which caused the test
    to fail with a very cryptic error message
    We fix this by explicitly allowing for status not being set yet.

[33mcommit f4a4e3d0eeec96e5ffdf869cc088908a5e3740d7[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Wed Feb 14 22:20:03 2018 +0800

    README: Add community section and quick links (#392)
    
    Add links to community section kubeflow/kubeflow
    
    Signed-off-by: Ce Gao <ce.gao@outlook.com>

[33mcommit 84adc4fe1d9a9e9317680079c232dd44e392c27d[m
Author: ChunTing Lin <sdf611097@gmail.com>
Date:   Wed Feb 14 10:33:25 2018 +0800

    Fix developer guide after move to kubeflow/tf-operator (#390)
    
    * Fix something after move to kubeflow/tf-operator
    
    * Revert release.py

[33mcommit fac13a650dfcbea3ebdb7bb9c17d4490164de670[m
Author: Jack <j5111261112@gmail.com>
Date:   Tue Feb 13 22:16:15 2018 +0800

    fix a typo in the README file. (#387)

[33mcommit b0561dd0b2e8254d09be71e106e2ea60727cffe9[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Tue Feb 13 02:22:05 2018 +0800

    *: Replace the repo name (#386)
    
    We have migrated the repo and we should replace the repo name with the new one.
    
    * Related to #350
    Signed-off-by: Ce Gao gaoce@caicloud.io

[33mcommit 36076742ef24b331f11a33a88a6cdfb8d90891f1[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sun Feb 11 19:55:07 2018 -0800

    Use ksonnet to easily define TFJobs to be run as tests (#374)
    
    * We want to easily run TFJobs corresponding to different TFJob specs (See #373).
    
    * We currently do this using Jinja2 templates and then having a python script test_runner.py to run those templates.
    
    * This PR migrates to using ksonnet to define those templates.
    
    * This PR is prework to running some TFJob E2E tests as part of ksonnet tests (kubeflow/kubeflow#207)
    
    Cleanup
    
    * Include smoke_tf.py inside the TFJob operator docker image; no reason we should have to build a separate image just to get that.

[33mcommit 75dab42eb1a11623411553f7be4763d6d06aa41c[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Mon Feb 12 10:38:34 2018 +0800

    travis: Add go build command in scripts section (#383)
    
    Close https://github.com/tensorflow/k8s/issues/382
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit bde716eec39a3be00c49defcbe0c0f9e20fc4b90[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Sat Feb 10 13:59:07 2018 +0800

    config.sh: Remove (#381)
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 57567df5019369c0ac38bc711aadad7b8cd6fa20[m
Author: Gaojin CAO <caogaojin@cmss.chinamobile.com>
Date:   Fri Feb 9 06:31:21 2018 +0800

    fix -version option: print version (#367)
    
    What this PR does / why we need it:
    this is a bug fix. if we using following command
    
    $ ./tf_operator -version
    it will not print any version information.
    
    Which issue(s) this PR fixes (optional, in fixes #(, fixes #<issue_number>, ...) format, will close the issue(s) when PR gets merged):
    NONE
    
    Special notes for your reviewer:
    NONE @jlewi
    
    Release note:
    
    NONE

[33mcommit 5557e87eb4956bff07ab5c5636d0708496d90224[m
Author: Jiayu Liu <Jimexist@users.noreply.github.com>
Date:   Fri Feb 9 06:05:14 2018 +0800

    nit: try to simplify e2e main.go (#359)
    
    I'm trying to simplify main.go files but this would better be taking baby steps.
    
    this very first step is to try to clean up spec.

[33mcommit 0de450f866b4af88e580eb59c01da7c65aacfddb[m
Author: Jose Aguirre <jose5918@gmail.com>
Date:   Wed Feb 7 12:22:03 2018 -0800

    Fix repo name env (#372)

[33mcommit 4220361234d78a98e9dcd0668861dfeb1c6c9e0e[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Mon Feb 5 21:10:07 2018 +0800

    controller.go: Fix a glog typo (#368)
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 55058f2138c640dae8412455256830c39ffaac81[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sun Feb 4 11:05:59 2018 -0800

    Use Argo rather than Airflow to run our E2E tests (#358)
    
    * Argo should be much easier to manage and use for our E2E tests
       Related to #205
       * In particular it should provide better isolation between test runs and easier access to logs
    * Create an Argo workflow to replace our Airflow workflow for running E2E tests
    * Create a ksonnet application to manage our E2E workflows.
    * Need to modify some of the steps in the workflow to activate the service account.
    * Update the Docker image used by our prow jobs to trigger the workflow rather than use Airflow.
    * Delete all the files related to E2E pipelines using Airflow.

[33mcommit e2b13e09c6cab08ae9b990f6e7f05826ac02e567[m
Author: Penghao Cen <cenph@caicloud.io>
Date:   Sat Feb 3 02:56:06 2018 +0800

    Deprecate MY_POD_NAME and use default namespace (#348)
    
    This PR fixes #341:
    
    Deprecate the ENV MY_POD_NAMESPACE and MY_POD_NAME
    And use NamespaceDefault by default

[33mcommit 95d9a3fd52e2897d56ddf90d57bb9ae8b543480d[m
Author: Jose Aguirre <jose5918@gmail.com>
Date:   Fri Feb 2 10:54:58 2018 -0800

    Fix local releaser (#361)
    
    Changes made:
    
    Moved version_tag to the common args to fix local release
    Intended to fix #360

[33mcommit 46767400bfe8624d53aa8960f78cca97310d9f33[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Sat Feb 3 02:54:10 2018 +0800

    *: Add copyright owner in go files (#364)
    
    * *: Add copyright owner in go files
    
    Close https://github.com/tensorflow/k8s/issues/266
    
    Signed-off-by: Ce Gao <gaoce@caicloud.io>

[33mcommit 330eb9239e006c2bcd04f90d57b01f873124c3e9[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Jan 31 05:12:15 2018 -0800

    Add an option to release.py to specify the tag for the image to use. (#357)
    
    * This will facilitate replacing Airflow with Argo because it will allow
      us to not rely on xcom to pass information between steps.
    
    * Instead we will just pick a label based on the prow environment variables.

[33mcommit cabc1c0ac9df89f2b8380533435b9c4f8c9859fc[m
Author: Jose Aguirre <jose5918@gmail.com>
Date:   Tue Jan 30 16:39:01 2018 -0800

    Fix helm test (#356)
    
    * Helm test was failing because validation for a tfjob required that replicaSpecs for a Parameter server specify a template. Helm test failure also was not reported.
    
    Changes made:
    
    * Updated e2e tests and examples to include a template for the PS replicaSpec
    * Check for None before concatenating the error.
    * Fixes #351 and #355

[33mcommit 7f97cf0f6aa4d7edfcc9fa4d80b553bdbdfff9c1[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Tue Jan 30 22:09:33 2018 +0800

    feat(group): Update to kubeflow.org (#354)
    
    * Fix #337
    
    * Per the discussion we've decided to use kubeflow.org as the group and not tensorflow.org. This PR makes all the relevant changes.

[33mcommit b0ea60efda06bcca1b90e019efa107a720e78741[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Mon Jan 29 21:52:09 2018 +0800

    WIP (#345)
    
    Close #281
    
    Separate CRD and controller
    Add CRD support in helm charts
    
    Signed-off-by: Ce Gao <ce.gao@outlook.com>

[33mcommit 8caa0c9cf71f2627dc02842b55d6c50e33641742[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Thu Jan 25 07:37:08 2018 -0500

    feat(dashboard): Namespace handling (#338)
    
    Fix #334 & #324.
    
    * Adds a selector to choose a namespace in which we should list the TFJobs.
    * By default we look into every namespace.
    screen shot 2018-01-19 at 4 10 56 pm
    
    * Also handles namespaces correctly at creation.
      If a job is created in a new (non-existant) namespace, the namespace will be created before-hand.

[33mcommit 2b8861489077d0966b76fe239490cde225c652b9[m
Author: Liang Mingqiang <secret.mqliang@gmail.com>
Date:   Thu Jan 25 07:35:45 2018 -0500

    replace TPR with CRD (#307)

[33mcommit 296b6523080653ed2e1b698cd4fef7ba38aa9604[m
Author: Penghao Cen <cenph@caicloud.io>
Date:   Thu Jan 25 20:34:10 2018 +0800

    Deprecate IsDefaultPS in TFJob CRD API (#343)
    
    * This PR fixes #329:
    
    * Remove IsDefaultPS in TFJob CRD API
    * Clean up hack/grpc_tensorflow_server/grpc_tensorflow_server.py which is useless
    * Update related docs

[33mcommit 11b2faddc8ced483d070038a23072b21e0eb59d5[m
Author: Jose Aguirre <jose5918@gmail.com>
Date:   Wed Jan 24 06:22:57 2018 -0800

    Update documentation (#342)
    
    Just updating docs on my first passthrough since I was going through them anyway.
    
    Changes made:
    
    Specify values.yaml file location
    Fix broken links
    Fix spelling
    Update tree

[33mcommit 74a958b06e9c3f2ac4e3a8ffb95c7114f53c925b[m
Merge: ca638ed8 d6a9510d
Author: Jingtian Peng <pjt73651@gmail.com>
Date:   Tue Jan 23 16:18:12 2018 +0800

    fix(convention): replace Tf with TF (#332)
    
    - Rename Tf to TF
    - More rename in docs

[33mcommit d6a9510ddf165a8257ff98ed878903e202d0fdae[m
Author: Penghao Cen <cenph@caicloud.io>
Date:   Fri Jan 19 18:17:48 2018 +0800

    More rename in docs

[33mcommit 002526863ff1dd79862173b425b63851c15ade49[m
Author: Penghao Cen <cenph@caicloud.io>
Date:   Fri Jan 19 13:41:47 2018 +0800

    Rename Tf to TF

[33mcommit ca638ed8c9364966055214a460aa188df6d9839e[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Mon Jan 22 20:46:45 2018 +0800

    Add recorder to publish event (#312)
    
    Add a recorder which can be used to publish Kubernetes events related to TFJob.
    
    Signed-off-by: Ce Gao <ce.gao@outlook.com>

[33mcommit c8b936817bc326013b533e144b792da52eaeedd2[m
Author: Penghao Cen <cenph@caicloud.io>
Date:   Sat Jan 20 10:05:26 2018 +0800

    Delete binary file (#331)

[33mcommit 2c0f7d49055f7eac2a5bdcd5bee8f5473db3b00f[m
Merge: 04425d99 291d3846
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Fri Jan 19 15:31:32 2018 -0500

    Merge pull request #336 from Jimexist/better-code
    
    feat(dashboard): better error handling in dashboard code

[33mcommit 291d3846f108b5aacfe5b4698ec295365f3863db[m
Author: Jiayu Liu <jiayu@caicloud.io>
Date:   Fri Jan 19 19:28:29 2018 +0800

    feat(dashboard): better error handling in dashboard code
    
    fixes #335

[33mcommit 04425d99e8b5ce0fca9f46d1be6a4f8b59b29309[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Jan 17 04:11:32 2018 -0800

    Take test failures into account when setting prow job status (#319)
    
    When setting prow job status, we want to look at the junit files and see if there were any test failures. Otherwise the k8s robot will report the tests as succeeding even when there were failures.
    
    Fix #315

[33mcommit 19e5e22917b4804f12ae9e2b24e7e1e04bcb9dca[m
Author: Connor Doyle <ConnorDoyle@users.noreply.github.com>
Date:   Tue Jan 16 21:56:02 2018 -0800

    Add service account name to dashboard if RBAC. (#298)

[33mcommit e4e12fecb370b261b9d308abaeac474b06d1ee0c[m
Author: Gaojin CAO <caogaojin@cmss.chinamobile.com>
Date:   Wed Jan 17 06:53:38 2018 +0800

    remove unused file rename.sh (#316)

[33mcommit 357a509213b24b1d960001861f85014b6ac28e51[m
Author: Penghao Cen <cenph@caicloud.io>
Date:   Tue Jan 16 22:22:55 2018 +0800

    Move around due to new directories layout (#273)
    
    As proposed here: #261
    
    This PR move folders around to match new directories layout with nothing changed.

[33mcommit b97dfc7d7737337ba026b6c2af149ec4cd3c8be0[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Jan 16 04:51:13 2018 -0800

    Fix a bunch of problems in TfJob CRD that crept in while tests were broken (#308)
    
    * In syncTfJob when checking whether a work queue item corresponds to a TrainingJob already in the map we need to check the UID. Otherwise we will not properly handle the case where a training job is deleted and then a new job is recreated with the same name.
    
    * We need to make sure that the Replicas field in TrainingJob is always properly set;
    
    * We were only initializing replicas in setup which was problematic in the case where the TfJob controller gets restarted because on restarted setup won't be invoked because the job is past that phase and as a result the replicas won't be reinitialized.
    
    * test_runner needs to ignore case when checking whether the job succeeded otherwise we conclude
    that successful jobs failed
    
    * The controller should only forget about job after the job has been cleaned up; not when it is marked as succeeded or failed.
    
    * Add back code to support termination policies use the worker and not the master as the chief
        *This was added in #221 and accidentally removed in the refactor in #234.

[33mcommit 77e272a1f85997705fca965ce4f3ea839bd0abed[m
Author: Gaojin CAO <caogaojin@cmss.chinamobile.com>
Date:   Tue Jan 16 01:58:23 2018 +0800

    fix broken link (#305)

[33mcommit 2109be90ddc733b784261fa937283513ff46cfb5[m
Author: Liang Mingqiang <secret.mqliang@gmail.com>
Date:   Tue Jan 16 02:55:29 2018 +0900

    add UpdateFunc to handle update events (#313)
    
    * Add an Update function to the controller.
    
    * The informer periodically generates an Update event but we aren't processing these because we don't have an update function.
    
    * As a result, TrainingJob.reconcile doesn't get called periodically and we aren't properly updating the status of the job.
    
    ref #309
    ref #293

[33mcommit 98a34a1b95b1aca10c5c0823f0c92eef78a5f939[m
Author: Jiayu Liu <Jimexist@users.noreply.github.com>
Date:   Sun Jan 14 06:43:01 2018 +0800

    feat(lint): use prettier and husky/lint-staged to enforce code style (#246)

[33mcommit 2eb5c7a30beeb034a1d86f5f06ed64543382255e[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sat Jan 13 08:59:28 2018 -0800

    Fix python lint checks (#303)
    
    * Some lint checks have crept in due to #274
    
    * Fix #298 (py lint issues)
    
    * Need to exclude some of the dependencies used by the frontend.
    
    * Fix util.run so that the output of commands prints out in airflow
    We need to redirect subprocess to a file and then read the file; nothing else seems to work.
    
    * Delete the test e2e_tests_dag_test.py; this was failing because of the way we import util and try to
    mock it out. Not worth fixing since we plan on getting rid of Airflow.

[33mcommit 91169fdbbfd8739477ab6dccb44e99e3d23f84c8[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Fri Jan 12 21:40:53 2018 +0800

    *: Fix API Version (#289)
    
    The API version from the event is empty, we should use the definition in apis, or we can not create the resources.
    
    Signed-off-by: Ce Gao <ce.gao@outlook.com>

[33mcommit 5cf32fa590d87937b48c4cd2d8672efc73e4c362[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Fri Jan 12 05:38:17 2018 -0800

    Fix setting defaults. (#299)
    
    * Need to register the default functions.
    * In setup we need to invoke setting the defaults on the objects.
    * This fixes a break introduced when we refactored the code.
    * Fix #284
    * Fix #297

[33mcommit e619ac4f596d80248db0f6152995d351e0db57be[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Thu Jan 11 14:45:03 2018 -0800

    The flag should be --controller-config-file. (#295)
    
    * The flag should be --controller-config-file.
    
    * This looks like a regression introduced by all the refactoring.
    
    * Fix #279

[33mcommit 833a25a3f00f0a87cd3d664942aa9111af702479[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Thu Jan 11 14:10:13 2018 -0800

    Fix the junit XML file format. (#291)
    
    * Fix the junit XML file format.
    
    * Test failures weren't being properly reported to gubernator because we had
      the wrong XML format. We need to use a nested failure tag and specify the
      message inside the tag.
    
    * Turn a failed test into a non-zero exit status.

[33mcommit 7ddafd516f347831bf2e428b72907f5f4bd82cf0[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Thu Jan 11 13:15:17 2018 +0800

    *: Implement the List interface for TfJobList (#278)
    
    The error is caused by the field name in TfJobList:
    
    // TfJobList is a list of TfJobs clusters.
    type TfJobList struct {
            metav1.TypeMeta `json:",inline"`
            // Standard list metadata
            // More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
            Metadata metav1.ListMeta `json:"metadata,omitempty"`
            // Items is a list of TfJobs
            Items []TfJob `json:"items"`
    }
    The name of field Metadata should be anonymous field.
    
    ERROR: logging before flag.Parse: E0110 16:52:24.998866    3707 runtime.go:66] Observed a panic: &errors.errorString{s:"*v1alpha1.TfJobList does not implement the List interface"} (*v1alpha1.TfJobList does not implement the List interface)

[33mcommit adc8ab956605e8167b3597dd63d338b3b08a2fe0[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Wed Jan 10 23:09:45 2018 +0800

    types.go: Fix CRDKind (#276)
    
    * The kind should be lower case; this error was surfaced while trying to implement the list interface #278
    
    Signed-off-by: Ce Gao <ce.gao@outlook.com>

[33mcommit e701fc58fd4dd30e0a466aeef5d3f5f99527b8d4[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Wed Jan 10 23:05:07 2018 +0800

    cmd: Fix the flag error caused by pflag (#277)
    
    * cmd: Fix flag error
    
    * Use flag library instead of pflag.

[33mcommit 9c93d5fe62e1b727039ac3ecb634488653cd1a48[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Mon Jan 8 11:57:35 2018 -0800

    Fix our continuous release process (#271)
    
    * Fix our continuous release process which is broken (#270)
       * Our releaser has been crash looping and not released any new images since 12/23.
    
    * Docker image used to build releases needs YARN and NodeJs to build the FE.
    
    * We refactored release.py and split cloning and building the code into two steps in #189. So we had to refactor how we build from the latest green to account for these changes.
       * release.py no longer has a function to periodically check the latest post submit and rebuild if
         necessary
       * Created a simple shell script launch_build.sh to check out the latest green commit and then build
         the code; We invoke the build function in the code checked out.

[33mcommit da1a861b4268bb64cd952522bae85986e1f7596d[m
Author: XsWack <xushiwei5@huawei.com>
Date:   Mon Jan 8 22:05:40 2018 +0800

    refactor controller logic (#234)
    
    This PR is mainly about refactoring the structure of controller keep style with kubernetes controller. And the main logic of package trainer is not modified. We can continue do refactor next step to make the controller stateless.

[33mcommit 1f85e46543904c8c2510c3883778ca7932d08ff4[m
Author: Liang Mingqiang <secret.mqliang@gmail.com>
Date:   Mon Jan 8 22:48:03 2018 +0900

    set faliures=true if failed deleting configmap (#272)

[33mcommit 13db0a5cc1c88824cd03fdb773977fb1e4bb6de8[m
Author: Christopher Beitel <cwbeitel@users.noreply.github.com>
Date:   Mon Jan 8 05:28:56 2018 -0800

    update initialClusterVersion to 1.7.11-gke.1 (#269)
    
    - set initialClusterVersion to 1.8.5-gke.0
    
    - Fixes #268

[33mcommit 190394dc7492d948d7664f7cd0a77680b7dcf4ee[m
Author: Jiayu Liu <Jimexist@users.noreply.github.com>
Date:   Sat Jan 6 03:19:05 2018 +0800

    feat(pipenv): Use pipenv to lock down python dependencies (#248)
    
    This is really a trivial change and so far I haven't been able to find time to adjust the docker file accordingly.

[33mcommit 5fdfdbf3d2a5ef205836ada3488620d4c3c73a51[m
Author: Penghao Cen <cenph@caicloud.io>
Date:   Fri Jan 5 23:30:16 2018 +0800

    Add proposed directories layout (#261)

[33mcommit f7803f1d3dd0c16a74a2ed248050f49917db8fbd[m
Author: Jiajin Zheng <zhengjiajin@caicloud.io>
Date:   Wed Jan 3 21:46:36 2018 +0800

    refactor dashboard backend, use versioned tfjob clientset (#258)

[33mcommit 2c8f325441c23385f4d279279ccfddd4f4754fe2[m
Author: Jiajin Zheng <zhengjiajin@caicloud.io>
Date:   Wed Jan 3 20:31:13 2018 +0800

    record event when rf_operator failover (#260)
    
    we should use real event recorder, then we can get event from tf_operator endpoint when tf_operator failover

[33mcommit 27930894a8fea07513f5b9fe6d88da9342b462d3[m
Author: Jiajin Zheng <zhengjiajin@caicloud.io>
Date:   Wed Jan 3 13:41:36 2018 +0800

    follow kubernetes flag convension (#259)

[33mcommit daf42d86d5e00de262b0e694da090f0ca79cfa80[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Jan 2 21:25:07 2018 -0800

    Misc Cleanup. (#262)
    
    * Fix a typo in a comment.
    
    * Remove the function run_lint in prow.py; this function is no longer used
      to run lint checks; there is a python script for that now.

[33mcommit 519b0acb32514f08a652ca69471f398f333980f1[m
Author: Jiayu Liu <Jimexist@users.noreply.github.com>
Date:   Wed Jan 3 11:38:29 2018 +0800

    Fix travis build errors due to lint issues (#257)
    
    * fixes #256 travis build failures
    
    * apply goimports -w to generated files to fix a bunch of lint issues.
    
    * There's no good way to tell whether a file was generated. a better way is to ensure that all files and newly submitted ones are goimported. also from now on I think it is better to enforce that even generated code are pre-goimported.
    
    * I didn't fix all the lint warnings because I am afraid of changing any code other than using goimport. I suggest that the original author do the changes instead.

[33mcommit ed7cae7552bacfa4d693fb8b5975d2e1ec62194a[m
Author: XsWack <xushiwei5@huawei.com>
Date:   Sat Dec 30 00:50:42 2017 +0800

    Refactor the TfJob to use K8s libraries  (#215)
    
    fix #206
    
    1.refactor the spec package to the kubernetes standard API, all detention of our CRD is in pkg pkg/apis/tensorflow and also the validation、 set Default for the CRD and some help func.
    
    2.for use K8s code gen, wo import the project https://github.com/kubernetes/code-generator, and some shell script in the folder hack. You can see how to use it in https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/
    
    3.All the file in pkg/client are all auto generated and we should not modify it manually.
    
    4.I do not plan to change the logical to use the sharedInformer and clientset. This PR is big enough, I think I can do it in a follow PR.
    
    5.SharedInformer is a helpful tool with cache that we can use it to list and watch our resources. More details you can see http://borismattijssen.github.io/articles/kubernetes-informers-controllers-reflectors-stores

[33mcommit 430cf179ba9c1ce4a134d3800f871dbbb0c73da1[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Fri Dec 29 08:49:13 2017 -0800

    Improve utilities for E2E tests. (#251)
    
    * Improve utilities for E2E tests.
    
    * Define a wrap_test function that takes care of some of the repitive work
      of configuring a TestCase object.
    
    * Define a TestSuite class to represent more than 1 TestCase.
      * This should make it easy to define a suite of TestCases ahead of time
        so that if we abort early, the junit file will still have an entry for
        all test cases even the ones that didn't get run.
    
    * airflow.py should set the logging formatter for the logger that logs to a
      file. The file handler doesn't appear to pick up the BasicConfig.

[33mcommit eb631ae7e697f1681ce626590101771b76ff3e84[m
Author: Jiayu Liu <Jimexist@users.noreply.github.com>
Date:   Sat Dec 30 00:48:09 2017 +0800

    fix(no-dup): reduce dup code in printVersion (#253)
    
    * fix(no-dup): reduce dup code in printVersion
    
    * fix tests

[33mcommit 433b234ce5a83f2b87cc76420b1ea34b1d4bad48[m
Author: Jiayu Liu <Jimexist@users.noreply.github.com>
Date:   Sat Dec 30 00:41:47 2017 +0800

    add gometaliner into travis build  (#254)
    
    and gradually tighten the linting rules. for now unused and golint are left out.
    
    some fixes are included in this PR.
    
    using gometaliner instead of just golint because:
    
    it covers more cases, e.g. the typo checkings.
    it also allows for disabling rules w.r.t regex, e.g. when you don't want to enforce all the comment rules enforced by golint
    
    * Part of #53

[33mcommit 420f9aaab4ef4d41cb49c3ff05b600c80decfd00[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Dec 27 12:37:27 2017 -0800

    Fix leaking of clusters in E2E tests #80 (#250)
    
    * setup_cluster needs to push the cluster name so that it is available to
      the teardown step before we try to setup the cluster so that the name
      is available even if setup_cluster fails.
    
    * setup_cluster also needs to handle the case where setup_cluster might
      already have been attempted; in which case we should reuse that cluster.
    
    * Fix #80

[33mcommit e98b10c640d67929931e4d87997218de2a575698[m
Author: Jiayu Liu <Jimexist@users.noreply.github.com>
Date:   Thu Dec 28 01:59:00 2017 +0800

    feat(lint): apply prettier to format frontend src/ code (#244)
    
    * feat(lint): use prettier and husky/lint-staged to enforce code style
    
    * do not lint markdown
    
    * feat(lint): apply prettier to code
    
    * some more small fixes
    
    * add badge, react plugin, etc.
    
    * Lint issues will be fixed in follow on PR.

[33mcommit 2634032763049e0b1a72bea08a118c1940fa7ed2[m
Author: Jiajin Zheng <zhengjiajin@caicloud.io>
Date:   Thu Dec 28 01:56:31 2017 +0800

    refactor code and format imported package (#245)
    
    * Define functions for printing the version and setting up signals.

[33mcommit a704246f170566ad33cb259028b68a7846700152[m
Author: Jiayu Liu <Jimexist@users.noreply.github.com>
Date:   Wed Dec 27 06:44:37 2017 +0800

    feature(lint): use prettier and lint-staged for frontend javascript code (#243)
    
    feature(lint): use prettier and lint-staged for frontend javascript code

[33mcommit 439b99f401ebbb451ac529a83683621f75a4a97a[m
Author: Jiajin Zheng <zhengjiajin@caicloud.io>
Date:   Wed Dec 27 06:43:00 2017 +0800

    add gitSHA into release (#227)
    
    * Bake the gitSHA into a constant in the go library
    * Addresses a TODO.

[33mcommit 60b8f7a20e99519f6cc1a1c9fe2bdae6444b1eed[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Mon Dec 25 10:17:31 2017 -0800

    Don't run glide install in travis builds. (#236)
    
    * We now check in the vendor directory so there shouldn't be any reason to run
      glide install in our travis builds.

[33mcommit bcfc14fc3d135b2fe7a0638a59f7f3d26858825a[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Fri Dec 22 20:09:26 2017 -0500

    Dashboard V0 for TfJob (#125)
    
    
    Addresses #57

[33mcommit 37af20da83e9af58a8eaff346b7609e0c168602b[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Fri Dec 22 15:59:30 2017 -0800

    Fix issues with tf_job_gpu test (#241)
    
    The setup cluster step should wait for the TfJob operator deployment to
    be ready.
    
    Ensure that all exceptions result in a failure message being reported to Gubernator.
    
    Upgrade and fix issues with Kubernetes py client 4.0.0; Fixes #242
    
    Bugs with gpu_test Fix #240

[33mcommit e108d556309a6b8921f33fe575d2931fe6ec1e39[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Thu Dec 21 16:36:20 2017 -0800

    Use the release/test python scripts pulled from the repo. (#237)
    
    The steps in our E2E tests should use the version of the test scripts
    at the revision that is being tested as opposed to the version baked into
    the container.
    
    Fixes #189

[33mcommit cb1e05389bfb25f21da84ca489eb8c8630f36b58[m
Author: lluunn <32310205+lluunn@users.noreply.github.com>
Date:   Wed Dec 20 06:56:17 2017 -0800

    allow using WORKER:0 as chief (#221)
    
    Allow training jobs to work without a master by treating one of the workers as the chiefs.
    
    * Fixes #192
    
    * This will allow TfJobs to be used with a lot of existing TensorFlow programs without modification since using worker 0 as the chief is a common pattern. Currently to run these programs using TfJob's  you need to spin up a dummy TensorFlow gRPC server just to serve as the master.
    
    * This is also necessary to support changes in estimator API with TF 1.4 (#61)

[33mcommit 6a2fc9c8e39931a09eb33d6e2f41804b27d9aed6[m
Author: Jiayu Liu <Jimexist@users.noreply.github.com>
Date:   Wed Dec 20 12:07:14 2017 +0800

    feat(coverage): add covealls support (#232)
    
    * feat(coverage): add covealls support to provide test coverage statistics
    
    * travis to Go 1.9
    
    * also add badge to README

[33mcommit 6d0455e0aaa0c3d21cc6454d7c08a8dc0b5ce608[m
Author: zhengjiajin <zhengjiajin@caicloud.io>
Date:   Wed Dec 20 12:05:42 2017 +0800

    let tfJob image configurable be configurable via the controller config(#228)
    
    * Fixes #207
    
    * test: add test for setDefault func

[33mcommit 924144955d17cdf4e90b582bcd86cd36d86ff810[m
Author: zhengjiajin <zhengjiajin@caicloud.io>
Date:   Mon Dec 18 22:22:40 2017 +0800

    use glide install --strip-vendor remove subpackage vendor (#231)
    
    * The strip vendor option removes the vendor subpackage in dependencies which avoids conflicts.

[33mcommit 5ada743ad4e5aa19caedc3c3367940324661c48e[m
Author: XsWack <xushiwei5@huawei.com>
Date:   Mon Dec 18 22:19:34 2017 +0800

     update k8s dependency to stable version (#230)
    
    * update k8s dependency to stable version
    
    * now the dependency for k8s is not the release version and this causes problems for #215, because the code-generator is outdated but the latest version for kubernetes 1.8.X fixes problem.

[33mcommit c8bcb9dca4eae1d8d2c31ff12895b7b51b78dd1c[m
Author: zhengjiajin <zhengjiajin@caicloud.io>
Date:   Mon Dec 18 09:05:23 2017 +0800

    remove subpackage vendor (#222)
    
    Add the vendor directory to the repository.
    
    * Checking in vendor is a common convention. For example, a variety of projects in kubernetes check in vendor.
    
       * This should help developers by allowing them to skip updating the deps and doing the manual step of fixing some of the dependencies.
    
    * remove subpackage vendor from .gitignore
    
    Examples of checking in vendor
    https://github.com/kubernetes/kubernetes
    https://github.com/kubernetes/client-go/tree/master/vendor
    https://github.com/ksonnet/ksonnet
    https://github.com/kubernetes/test-infra
    
    Some counter examples
    https://github.com/grpc/grpc-go
    https://github.com/GoogleCloudPlatform/google-cloud-go
    https://github.com/google/glog
    https://github.com/kubernetes/helm
    
    It looks like our total repo size is ~125 MB and ~110 MB of this vendor.
    But you have to download all 125 MB if you want to fully build it; so a smaller repo size doesn't really make cloning and building the code any faster.
    
    So following the K8s convention and checking in vendor makes sense.

[33mcommit eb0fd5f31ea2d1f5d40e0b8c26718ffad2081b28[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Thu Dec 14 20:40:03 2017 -0800

    Set state to failed if there is a problem initializing job (#219)
    
    * Fixes #218
    * If a job can't be initialized (e.g. because the spec is invalid) the state should be set to failed.
    * I refactored the code so that in the event that the call to setup fails, it will end up getting retried as part of the next attempt at reconcilation
    * I moved the call to setup into run
    * We also refactored the code for reconciling the state into a separate function reconcile. I did this because we want to call reconcile when run is first invoked and not wait for the reconcile interval to elapse.
    * Delete the method triggerCreatePhase; I moved the relevant code for setting the phase into setup.
    * Delete some code that is a legacy of the etcd-operator and no longer being used.

[33mcommit 0bd02ace4cda7cc050654e6af572f98a4c0c9e6a[m
Author: zhengjiajin <zhengjiajin@caicloud.io>
Date:   Fri Dec 15 03:23:53 2017 +0800

    replace tf-job-operator-config configmap when it already exist (#225)

[33mcommit 7f3992259b1aa27e98460f3967b7fd27047318fa[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Fri Dec 15 03:21:26 2017 +0800

    controller.go: Fix a print error (#226)
    
    Signed-off-by: Ce Gao <ce.gao@outlook.com>

[33mcommit 2b67180b1f52eb894fff37c9ab33063b152f2107[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Dec 12 17:42:42 2017 -0800

    On GKE mounting volumes should no longer be required for GPUs. (#217)
    
    * GKE uses device plugins so we should no longer need to configure the TfJob
      controller to mount files from the host.
    
    * So we can remove entries from the config map.

[33mcommit c290c0dcf7008096a92253041e0df78b2dd31ca6[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Dec 12 16:29:02 2017 -0800

    Fix issue with handling of json errors. (#220)
    
    * Fix issue with handling of json errors.
    
    * Always run the teardown step even if there are upstream failures.

[33mcommit c2598e8cfb37a2ba99416179a19053c9763f080b[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Dec 12 11:02:22 2017 -0800

    Add a basic GPU job test as part of our E2E tests. (#213)
    
    * Add a basic GPU job test as part of our E2E tests.
    
    * Fixes #164
    
    * Update our E2E test infrastructure to spawn GPU clusters.

[33mcommit 7167512fc8d1545169fcdda82f45bce3fc69f9df[m
Author: Deyuan Deng <deyuan@caicloud.io>
Date:   Wed Dec 13 02:30:19 2017 +0800

    update developer guide (#216)

[33mcommit 08ec97f82d677e54e08096bd3e72c3eafd53c04d[m
Author: lluunn <32310205+lluunn@users.noreply.github.com>
Date:   Mon Dec 11 13:47:50 2017 -0800

    Add terminationPolicy to TfJobSpec (#204)
    
    * First PR to fix #192 (Don't require master or chief.)
    
    * add terminationPolicy to TfjobSpec
    
    * terminationPolicy should default to using a chief corresponding to the master.

[33mcommit 76fcb3d67b9e3fbb105774cc266471ef34730efd[m
Author: cbockman <c.bockman@gmail.com>
Date:   Mon Dec 11 13:44:55 2017 -0800

    minor spelling porxy => proxy (#211)

[33mcommit f026676351b86ef68ead48e06856d3e7536dfadf[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Dec 6 16:38:35 2017 -0800

    Split cloning the repo and building the images into two steps in our airflow pipeline (#200)
    
    * Fixes #189
    
    * We want cloning the repo to be its own step because for all subsequent steps we want to use the code pulled from the repo so that we can test the code in presubmits.

[33mcommit ff619574b79972cc53331a9895d533087d0e4873[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Mon Dec 4 13:30:45 2017 -0800

    Create separate commands to clone and build the repo (#199)
    
    The goal is to be able to split cloning the repo and building the artifacts into two steps #189
    
    We want cloning the repo to be a separate step because we want to pull release.py from the repo when actually building the code. But in order to invoke release.py from the repo we first have to check it out.
    
    In a follow on PR we will update the E2E test pipeline to use the new commands.

[33mcommit e4a436da92e198dcb88c89c33010608e0c8a23bf[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sun Dec 3 20:57:38 2017 -0800

    Install yarn and nodejs inside the Airflow container. (#198)
    
    * These will be used to build the Dashboard.
    
    * We split these changes off from 125 because they need to be submitted
      first and Airflow rebuilt in order to get the presubmits for #125 to pass.

[33mcommit 03c5b4ba68200b96c6bf98cb7a00a4513fee9f2e[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sun Dec 3 20:35:57 2017 -0800

    Fix helm templates so that we don't require a configmap.  (#176)
    
    * Fix helm templates so that we don't require a configmap.
    
    * If no --cloud isn't specified then we shouldn't specify
      controller_config_file
    
    * Add options to allow the user to specify the name of a config map
      if they want to set the configmap themselves.
    
    * The options are set so that specifying a custom configmap overrides
      the value selected by setting cloud so that the two options can be used
      together.
    
    * Update the README.md

[33mcommit 6fc7def7f9c429cd83ddec5a6a8497b08f53f5c8[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sun Dec 3 20:35:11 2017 -0800

    Add an option to build Docker images with GCB. (#187)
    
    * Building with GCB can be much faster when using GCP and GCR because we
      can avoid pushing and pulling images to our local machine.

[33mcommit 8800de58c5a515d25dd18ce917b6f98e7dd6e1da[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sun Dec 3 20:33:18 2017 -0800

    Update the Airflow deployment to use Docker images built from a clean tree. (#197)
    
    This fixes the broken presubmits #196.

[33mcommit ab95d3594552b083d33fcff64535d246d1f1755c[m
Author: Mike Wilson <hyperbolic2346@users.noreply.github.com>
Date:   Sat Dec 2 14:59:54 2017 -0500

    Fixing front page documentation to have grpcServerFilePath (#190)

[33mcommit c2e8fef6b6d112d67b854cf44ad37846d8dd3684[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Sat Dec 2 14:59:18 2017 -0500

    fix cuda issue on Azure (#194)

[33mcommit 84241c26826343d52b48b04cd39c63ac64cdbabb[m
Author: Jingtian Peng <pjt73651@gmail.com>
Date:   Wed Nov 29 11:34:15 2017 +0800

    replace deprecated tf.initialize_all_variables with tf.global_variables_initializer (#184)

[33mcommit 0b8eb7081425a75226138a61b5b9dc2ba67aecf1[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Wed Nov 29 11:33:56 2017 +0800

    build_and_push.py: Support python3 (#183)
    
    In python 3 we have to use decode to explicitly convert from bytes to string.

[33mcommit f8ec7624e4859fa86b9cc393fb44207dd513ce8c[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Wed Nov 29 11:33:05 2017 +0800

    tf_job_design_doc: Fix the apiVersion (#182)
    
    Signed-off-by: Ce Gao <ce.gao@outlook.com>

[33mcommit e810d220e3781b38ad69f2a6c97e622f83ff60ff[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Wed Nov 29 10:47:08 2017 +0800

    py: Add requirements.txt (#180)

[33mcommit 14f3c0bd8384ebad39068570b35b89d17ccfa126[m
Author: Jingtian Peng <pjt73651@gmail.com>
Date:   Tue Nov 28 10:57:26 2017 +0800

    resolve a merge conflict imported by commit ae8c31 (#178)
    
    * The conflicts are in files used by our Airflow deployment and aren't tested by our E2E tests which is why our tests are passing.

[33mcommit a116941a15d5135d7904ed69ea4727ff8dab62b8[m
Author: Ce Gao <ce.gao@outlook.com>
Date:   Tue Nov 28 10:38:23 2017 +0800

    tf_job_design_doc.md: Fix a typo (#177)
    
    https://github.com/jlewi/mlkube.io/ is moved to this repository but the link is not updated.

[33mcommit 16bf1e4cce3eb6e38983aa35c34cb002b84c4c5b[m
Author: Jingtian Peng <pjt73651@gmail.com>
Date:   Mon Nov 27 00:05:51 2017 +0800

    replace Google and Golang repos with corresponding github repos (#172)
    
    Fix #171 .
    
    To better accommodate Chinese contributors we use mirrors of golang.org and google in Github because the former are blocked in China but Github is not.

[33mcommit ae8c3182fb47b65e84eb6e1ac3b5d5fe5be7b128[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sat Nov 25 10:52:16 2017 -0800

    Tooling to make it easier to run a bunch of TfJob tests. (#168)
    
    Tooling to make it easier to run a bunch of TfJob tests.
    
    * We'd like to be able to add tests just by adding a TfJob spec in a YAML
    file.
    
    * test_runner.py provides a simply binary that can take a TfJob spec in a YAML
    file, run it, and perform some basic validation checks.
    
    * tf_job_client is some helper routines for submitting and managing TfJobs;
    basically a rudimentary Python client library for TfJob.
    
    * In a subsequent PR we will integrate this with our continuous test
    infrastructure.
    
    * This is related to #164 (E2E test for GPUs). The goal is to extend the test infrastructure to make it easy to add more tests.

[33mcommit 990a79148a7d7cf658e689c37fbd9578ae155e09[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sat Nov 25 09:03:19 2017 -0800

    Run python lint and unittests as part of our E2E test pipeline (#166)
    
    * Create a binary to run python lint and unittests .
    
    * This will be used to run lint and unittests as part of the E2E pipeline.
    
    * Integrate python lint and unittests into our E2E pipeline.
    
    * Update the deployment.
    
    * Fix #126 (lint results not reported in gubernator)
    * Fix #53 (run pylint as a presubmit)
    * Fix #101 (run python unittests as part of pre/post submit)
    
    * To support this change we modify our Airflow E2E pipeline so that it will checkout the source to a directory that is on our PD and passes that directory to subsequent steps via XCom.

[33mcommit b48958b02d69bd474a3651ece058d6dc8ef94b18[m
Author: Haitao Chen <haitch@users.noreply.github.com>
Date:   Wed Nov 22 10:53:09 2017 -0800

    Stop hard coding the namespace in the config map (#169)
    
    The namespace can be specified using helm install --namespace
    
    We don't set the namespace for the TfJob controller; so forcing the namespace for the configmap was a bug.

[33mcommit 29df6b43f983946f56dccdedaa03c0fde28ce86d[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Nov 22 08:38:21 2017 -0800

    Create a binary to run python lint and unittests . (#163)
    
    Create a binary to run pylint and python unittests.
    
    This will be used in our E2E test pipeline.
    
    Partial fix to #126 (lint results not reported in gubernator)
    
    This binary creates junit files with junit files
    Partial fix to #53 (run pylint as a presubmit)
    
    This provides a binary to run lint that will be incorporated into our E2E pipeline.
    Partial fix to #101 (run python unittests as part of pre/post submit)
    
    This PR provides a binary to run the unittests; subsequent PR will integrate it into Airflow.

[33mcommit f4c694dd2b0f2c819981e00405a6146fbdd2e867[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Mon Nov 20 22:25:44 2017 -0800

    Integrate Airflow with Prow (#158)
    
    Our PROW jobs should trigger our Airflow pipelines to run our tests.
    
    This aims to address #120
    
    This PR changes the code invoked by PROW to trigger Airflow.
    
    We replace bootstrap.py with py/airflow.py; we use this Python script to trigger an Airflow pipeline.
    
    We update the steps in the Airflow E2E pipeline to create junit XML files in GCS so Gubernator can compute the results of the test.
    
    This PR introduces a regression into our PROW jobs in that the current pipeline doesn't run lint checks on our Python code; we will fix that in a subsequent PR.
    
    Our Prow job uses the same base container as our Airflow deployment; the only difference is the entrypoint.
    
    We get rid of main.go because our AIrflow pipeline takes care of setting up the CRD and cluster.

[33mcommit fee4a87283c9600e18cc354a04230d81e493cf07[m
Author: lluunn <32310205+lluunn@users.noreply.github.com>
Date:   Mon Nov 20 22:03:26 2017 -0800

    fix dev guilde (#162)

[33mcommit a7b9693b174f9ba02ea33d51d4d94e66d8eca7cc[m
Author: huangyue <yue.huang.enst@gmail.com>
Date:   Sat Nov 18 03:12:15 2017 +0800

    change jobname from task-runtimeid-index to jobname-task-runtimeid-index (#151)
    
    * change jobname from task-runtimeid-index to jobname-task-runtimeid-index
    
    * tfjobname truncated to 40 characters
    
    * fix typo in comments

[33mcommit 35603cde99a60be1facaa782baf79d586e160e66[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Fri Nov 17 05:41:32 2017 -0800

    Airflow pipeline to run our tests (#144)
    
    Create an Airflow pipeline to run E2E tests in parallel.
    
    * The high level goal is outlined in #120.
    
    * This commit creates an Airflow pipeline that is equivalent to our existing test; it has steps to build the artifacts, setup a GKE cluster, run the helm test and teardown the cluster.
    
    * We create a suitable Dockerfile and K8s deployment for running Airflow on K8s.
    
    * The README.md provides instructions.

[33mcommit c4ef91f1c3d24b0b049edb5a02fb1c4abc2f25bb[m
Author: huangyue <yue.huang.enst@gmail.com>
Date:   Thu Nov 16 22:31:16 2017 +0800

    rename jlewi/mlkube.io in glide.yaml (#153)

[33mcommit db5c875455bc323a8215785053e792d3206c4c4a[m
Author: huangyue <yue.huang.enst@gmail.com>
Date:   Thu Nov 16 13:33:09 2017 +0800

    add Create(), Delete() in TfJobClient interface, rename tpr_util.go to tf_job_client.go (#152)
    
    Add routines to make the go client more complete.
    
    Rename a poorly named file.

[33mcommit 397ef28f38b2c515c139f4f736d0e38aec78e00f[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Nov 14 17:33:24 2017 -0800

    Create binaries to run steps in an E2E test pipeline. (#148)
    
    * deploy provides commands to provision resources (K8s clusters) needed
      for E2E tests.
    
    * release.py make some changes needed to build artifacts as part of an
      E2E pipeline.
    
    * util.py add a work around for an issue with the Kubernetes Python client
      library that prevents credentials from working correctly when using a
      service account.
    
    * test_util provides routines for creating junit XML files; this will be
      used to create results for gubernator.
    
    * These binaries will be used in #120

[33mcommit 24412eafe18e717a0322254c90500e4347d134fa[m
Author: huangyue <yue.huang.enst@gmail.com>
Date:   Wed Nov 15 09:31:38 2017 +0800

    rename InClusterConfig() to GetClusterConfig() (#137)
    
    use env var KUBECONFIG to retrieve kubeconfig (same as kubectl)

[33mcommit a99843645774dcd570d93f7f3dfa4d50ccaeee69[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Nov 14 16:08:08 2017 -0800

    Fix a typo in the command line help. (#147)

[33mcommit 3b436d3f6fe31b763e5641b0397a6c7bdc800df5[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Nov 14 15:47:00 2017 -0800

    ignore too-many-locals. (#146)

[33mcommit f27b8de7192cc359a3a07b8afa2f497ef1c932cb[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Nov 14 15:22:55 2017 -0800

    Turn release.py into a binary to build the artifacts for all the different contexts (#133)
    
    * Turn release.py into a binary to build the artifacts for all the different modes.
    
    * We have multiple contexts in which we want to build the images and other artifacts
           * local
           * Tests
           * releases
    * This CL changes release.py so handle all three cases.
      * This increases code reuse and provides greater consistency.
      * Prior to this CL building the operator image didn't build other artifacts;
        notably the helm package.
    
    * This change is intended to facilitate transitioning to a workflow system
      for running more complex test and release workflows.
      * Currently the test infrastructure is using runner.py to build the
        images. The tests don't actually build the helm package.
      * We want to replace runner.py with a binary (release.py) that can
        be invoked to build the artifacts. This way our test/release workflow
        can just invoke that binary.

[33mcommit ad34ef2f8af8c245d92e2f0ae791d87a8b2dd812[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Nov 14 15:07:38 2017 -0800

    On RBAC clusters, test needs a service account with appropriate permissions. (#145)
    
    * On RBAC clusters our test needs to run with a service account that can
      create jobs and get pods.
    
    * So we run our test using the tf-job-operator service account since that
      service account has the requisite permissions.

[33mcommit 17a8dcc13b772c376e6dc80a3a6bbafb579cf841[m
Author: huangyue <yue.huang.enst@gmail.com>
Date:   Tue Nov 14 14:29:59 2017 -0600

    rename clus to tfjob in controller.go, which is a bad name from etcd-operator (#138)

[33mcommit 2d162e5d1da86e4800b99bfe0abe4dd1dffe110b[m
Author: huangyue <yue.huang.enst@gmail.com>
Date:   Tue Nov 14 14:29:23 2017 -0600

    fix a log issue (#141)
    
    Should be using log.Fatalf not log.Fatal

[33mcommit bf7556d3b7029f322b61d792a8330da0558f5d9e[m
Author: Enhua Li <lienhua34@163.com>
Date:   Tue Nov 14 14:18:17 2017 -0600

    fix(*): amend the number of worker and ps in example yaml spec for a distributed job (#142)
    
    There is a conflict about the number of worker and PS between the yaml and figure comment.

[33mcommit 440876843321fa56c699270934ff3740b75a1cac[m
Author: Penghao Cen <cenph@caicloud.io>
Date:   Fri Nov 10 12:21:30 2017 -0600

    Remove trailing slash of host (#134)

[33mcommit 46a4273da96f4b79dc0f607453aed661fcb36190[m
Author: Penghao Cen <cenph@caicloud.io>
Date:   Fri Nov 10 00:34:03 2017 +0800

    Minor fix typo and redundancy (#131)

[33mcommit 214020a818cb65ccc8567f92a7f0557dd87bda81[m
Author: Jiayu Liu <Jimexist@users.noreply.github.com>
Date:   Thu Nov 9 09:50:57 2017 -0600

    Update developer_guide.md (#129)
    
    update some markdown format

[33mcommit ead44b0680bb6211bf949ce393c88ab299cd064d[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Mon Nov 6 19:32:09 2017 -0800

    Use K8s Garbage Collection (#127)
    
    Turn on garbage collection to ensure all resources are properly deleted when the TfJob is deleted.
       * This fixes #42 (Use K8s GC) and should fix #107 (TensorBoard replicas not properly deleted).
       * Relying on GC to delete resources when TfJob is deleted as opposed to deleting them ourselves
         is the recommended approach (see #42)
      * We don't remove the deletion code because per #128 we will start deleting resources when the
        job is done rather than waiting for the TfJob to be deleted.
    
    * To turn on GC we need to set owner references on all resources created for TfJob.
    
    * Remove garbage collection and rely on K8s garbage collection.
    * Garbage collection was inherited from the etcd-operator which we copied.
    * Prior to K8s 1.8 I don't think K8s fully supported Garbage Collection for CRD.
    * But with K8s 1.8 GC should be fully supported so we can rely on K8s GC rather
      than implementing our own.
         * This introduces a dependency on K8s 1.8 since garbage collection for CRDs isn't supported in
         earlier versions.
    
    * Update cleanup_clusters.sh
    * Delete redeploy_operator.sh; we use helm now.
    
    * Update the readme to indicate GKE 1.8 is required since we depend on K8s garbage
    collection support for CRDs.

[33mcommit a08bda95a541fdd2fd0dd56d77e8e336e1567f4f[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sun Nov 5 15:53:30 2017 -0800

    More verbose logging of resource deletion (#124)
    
    Increase logging to try to track down issues with TensorBoard not being deleted #107.
    
    Set the default image in the helm chart to the latest image
       * We add the latest tag with every release
       * Furthermore, when we generate releases we actually override values.yaml to pin values for certain images.
    
    Update e2e.go so that we can potentially run multiple jobs to try to stress the operator.

[33mcommit e3809b4859c48d9ea5488bb0a2d5b8c77ff42c0d[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sun Nov 5 11:30:23 2017 -0800

    Fix rbac settings in chart. (#123)
    
    * TfJob operator needs permissions to create configmaps because we create
      a configmap for the PS in order to create a std gRPC server.
    
    * Change the source repository mentioned in the chart.

[33mcommit f397d0ad226b28b1520f461c5bd4e7881c34d3b3[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Fri Nov 3 12:20:34 2017 -0700

    [WIP] Notebook demonstrating use of TfJob on GKE (#102)
    
    * Start a notebook to run some of the TensorFlow examples.
    
    * Got a minimal example submitting a job and getting the status.
    
    * Delete checkpoint.
    
    * Rename go packages from jlewi/mlkube.io > tensorflow/k8s.
    
    * Hacking on the example. Commit before pulling in latest changes.
    
    * * Make build_and_push a function
    * Notebook should build docker images
    * Launch K8s batch job to download data.
    
    * Add and set environment in TF_CONFIG.
    
    * Training is working with fixes to the operator.
    
    * Update build_and_push_image to stream logs so we get output while its running.
    
    * More work on the example.
    
    * Add links to TensorBoard
    * Provide information about getting the logs.
    
    * * Headings at 2nd level (##) are rendering in GitHub. Check if this is
      because they are part of a cell with other markdown.
    
    * update the TOC.
    
    * Try changing level of Requirements to ### to see if that causes it to render in GitHub.
    
    * Expand the markdown comments.
    
    * Try removing the TOC to see if that helps render markdown in GitHub.
    
    * Latest changes to the notebook.
    
    * latest.
    
    * Start work on a test for the notebook.
    
    * Latest edits.
    
    * Move a lot of the code into util.py.
    
    * There's a problem with 1.8 clusters because rbac isn't setup correctly.
    
    * Notebook is working.
    
    * Fix some bugs in the test and notebook.
    
    * Fix some issues.
    
    * Fix lint issues.
    
    * Need to install the Kubernetes python client library in the bootstrap image.

[33mcommit 5a6991541123792b9bb6f135ac332084c107dba8[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Fri Nov 3 15:10:51 2017 -0400

    Fix issue in tpr_util.Delete() (#121)
    
    Currently, calling Delete() delete all the jobs in the namespace, even though a name is passed as argument.

[33mcommit 2e340348baad1bbfea7d54d00008f1393cba39f3[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Fri Nov 3 05:42:33 2017 -0700

    Tag docker images with "latest". (#119)
    
    Fixes #116

[33mcommit 51ea8fcde9ce0291a0ac504952804aa3a11f91b6[m
Author: Sertaç Özercan <sozercan@gmail.com>
Date:   Thu Nov 2 19:47:55 2017 -0700

    update apigroup in the chart (#117)
    
    Fixes #118

[33mcommit 17f204a9889207b2463384d71e9cec4ad6bab042[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Thu Nov 2 06:04:04 2017 -0700

    Helm instructions (#111)
    
    * Add links describing how to configure service accounts for tiller if needed.
    
    * Update the link to helm.

[33mcommit e8070688034dd1af66c3527518573127da9f6825[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Nov 1 13:49:40 2017 -0700

    Name label (#105)
    
    Add Name of the job to the labels.
    
    This will make it easier to fetch resources belonging to a specific job.
    
    * Fixes #72.

[33mcommit 083bf79aa90c18945355f7e314c57e04a6280b7e[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Nov 1 08:54:25 2017 -0700

    Change group to tensorflow.org and version to v1alpha1. (#103)
    
    * We have abandoned the name mlkube.io and moved the code into tensorflow
      so it makes sense to use tensorflow.org as the group.
    
    * We downgrade the API version from v1beta1 to v1alpha1.
      We want to follow the guidelines used by K8s for alpha, beta, stable:
      https://github.com/kubernetes/community/blob/master/contributors/devel/api_changes.md#alpha-beta-and-stable-versions
    
      according to this definition we our closer to alpha since the API hasn't
      achieved stability.
    
    Fixes #85

[33mcommit 2b425bde35233657ca590a18f65403b2707bcae2[m
Author: Sertaç Özercan <sozercan@gmail.com>
Date:   Mon Oct 30 18:45:29 2017 -0700

    Update helm install syntax in readme (#104)

[33mcommit bc0c348ea4c7844295992d162bc1b66264b68258[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Mon Oct 30 12:33:32 2017 -0700

     Fix bugs in the release script. (#100)
    
    Need to set GOPATH properly before building the code.
    
    Copy the release scripts to the Docker image rather than cloning the
    repo. This makes it easier to push changes to the releaser because
    we can just build the image with local changes. Otherwise we need to
    commit the changes first and then build the Docker image.
    
    Need to break out of the loop if only building once
    
    Need to install helm in the releaser image because we use it to build the helm package.
    
    Need to build the docker images using GCB so we don't need to access docker inside the pod running the releaser.
    
    Store the image version in a json file inside the container and print out.
    
    Related to #63

[33mcommit 7539bc810ea48fd6d072dac23e6def6332c70c56[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sun Oct 29 20:11:20 2017 -0700

    Fix bugs in the release script. (#99)

[33mcommit 7fce996ca09f23f92f156c362835b30b8d0e85f5[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sun Oct 29 19:16:22 2017 -0700

    Update release.py so we can run it continuously. (#98)
    
    * release.py will regularly check for postsubmit results and cut new
      releases.
    
    * Create a Docker image to run and it a Kubernetes spec file to deploy it
      on a K8s cluster.

[33mcommit 4b7b1fe15139e3a3b2ade9ee7242204dbe10f2bc[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sun Oct 29 17:08:55 2017 -0700

    Need to set environment to enable Estimators with TF <=1.3  (#94)
    
    * Add and set environment in TF_CONFIG.
    
    * Need to set environment to enable Estimators with TF <=1.3
    
    * RunConfig (part of Estimator API) depends on the environment being
      set in TF_CONFIG to 'cloud'. If we don't set environment to 'cloud'
      then is_chief isn't enabled for the master and the TF program doesn't
      make progress.
    
    * TF 1.4 should be backwards compatible; i.e. it doesn't make use of
      environment in TF_CONFIG. So this code should be forward compatible.
    
    * Expand the unittest for replicas.go to cover the new code.
    
    * In replicas.go we should make a full DeepCopy of the PodTemplateSpec
      before modifying it. It turns out K8s provides auto-generated code to
      perform deep copies.

[33mcommit 4369c2cd1ddfaa4aac69c093ea3380d187adf070[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sun Oct 29 16:04:00 2017 -0700

    Fix the E2E test by specifying cloud when deploying the helm package. (#97)
    
    * Fix the E2E test by specifying cloud when deploying the helm package.
    
    * We need to specify the cloud flag when deploying the helm package so
      the a volume is properly created with the TfJob config.
    
      * This should fix issue/96
    
    * run functions in util.py should capture stderr in addition to stdout so
      we get the output from help-test
      * This should fix issue/95

[33mcommit 16fc3d0e77215a762b2e562b08a656d894cbc473[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Thu Oct 26 17:00:25 2017 -0700

    Add python lint check to travis and fix python lint issues (#91)
    
    Add a python lint check
       Partial fix to #53
    
    Capture stdout/stderr of the test and upload to GCS so we have a proper build log that outlives the pod lifetime.
      Fixes #82
    
    * Run autopep8 on all py files.
    
    * Fix python lint issues
    
    * pylintrc was copied from kuberntes/test-infra
    
    * Disable import-error message since some py files run inside containers
    which will have different libraries then the machine running the lint check.

[33mcommit 6d8f9f9851ec6e02923215edf331873e27df8b53[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Thu Oct 26 15:41:13 2017 +0100

    #71 Simplify accelerators config (#90)
    
    * #71 simplify accelerators config
    
    * Add a cloud value that accepts gke or azure and will create the corresponding ConfigMap
    
    * Users can create the config manually and not specify a cloud option in order to have complete control of the config.

[33mcommit ed770b4c8e4cd7b4c1f3934519d8001b90dda733[m
Author: Jiayu Liu <Jimexist@users.noreply.github.com>
Date:   Wed Oct 25 22:10:05 2017 -0500

    Update README.md (#92)
    
    to fix a small typo found during doc reading

[33mcommit bcf2f086f8c6c41e9467f8260581b06ec3bb3a77[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Oct 25 16:22:42 2017 -0700

    Update test infrastructure to use repo tensorflow/k8s (#87)
    
    * Update test infrastructure to use repo tensorflow/k8s
    
    * Update README to refer to new repo.
    
    * Rename go packages from jlewi/mlkube.io > tensorflow/k8s.
    
    * Update repo location in the helm e2e test.
    
    * Update links.
    
    * Update the readme.md

[33mcommit 1332e8a9b889530d249dc112124c86812ed23287[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Oct 24 10:21:43 2017 -0700

    Create symbolic links in GCS to output of presubmit results. (#79)
    
    * Create symbolic links in GCS to output of presubmit results.

[33mcommit 2774d51791b937bd4438a2c9ca212ee1c651d2c3[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Oct 24 05:38:28 2017 -0700

    Fix periodic results (#76) (#78)
    
    * Fix results for periodic test.
    
    * We should only add "pull" if PULL_REFS is non-empty; there was a bug in
      previous commit.
    
    * Add a unittest to ensure we correctly create started.json in the case of
    periodic tests.

[33mcommit b465c81392b7c167469c29ca3fa00ca7731d298b[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Oct 24 05:38:06 2017 -0700

    Release scripts (#69)
    
    Scripts to release artifacts.
      * Find the latest green postsubmit
       * Build and push docker images and helm packages
       * Helm packages are published on GCS and are installable via URL, so users don't need to clone the repo to deploy it.
    
    * build_and_push.py should be sure to execute the call to get git sha in
      the root.
    
    * organize py scripts ina new directory to facilitate reusing libraries.

[33mcommit 4887cfdf67cedab67b022594bc7b7301a5310674[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Oct 24 04:23:56 2017 -0700

    Overhaul the documentation (#73)
    
    * Overhaul the documentation
    
    * Create a separate design doc, tf_job_design_doc.md that more fully
      explains the design and motivation.
    * Create a separate developer guide with information about building the
      repository and other information specific to developers/contributors.
    
    * Rewrite README.md to focus on information relevant to users of TfJob.
      Add a lot more information about how to monitor jobs as well as information
      explaining how it works.
    
    * * Instructions should install package from URL.

[33mcommit 3e19cf5f9b58be5e5c537389e1ba9e1ed1120714[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Mon Oct 23 21:19:44 2017 -0700

    Another attempt to fix periodic jobs. (#77)
    
    * GCS output for periodic jobs appears to be in kubernetes-jenkins/logs/mlkube-build-periodic
    * Don't set PULL in started.json if its an empty string; this is causing the
      parse errors.
    * Try to fix issue #76

[33mcommit 7913375113ca996aa78c9e2cceb11730214d08bd[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Mon Oct 23 16:10:40 2017 -0700

    Fix location of the post submit results. (#74)
    
    * GCS location needs to include github organization and name.

[33mcommit a1d0405d0941f5b10addb7548f8a8a596a917472[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Thu Oct 19 21:50:31 2017 -0700

    Update the tfjob image to the latest green CL.
    
    * The sample jobs were broken because they were using the default PS
      but the chart hadn't been updated.

[33mcommit 7a05c260c9a0e9b43bd411dac0e733d97adbb514[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Thu Oct 19 16:25:04 2017 -0700

    Record latest green from postsubmit (#66)
    
    * Write a json file to GCS with the results of the test on success.
    This way we can keep track of the latest green results.
    * This will be used to cut releases on the latest green CL.

[33mcommit ca33d3d8c5dfbfc288a116d7cdde4eb60629ccbb[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Thu Oct 19 20:38:10 2017 +0200

    Parameter Server: Run TF server by default (#36)
    
    Add an option to automatically run a standard gRPC server for parameter servers so users don't have to supply it themselves.
    
    * Use a ConfigMap to inject a python script that can start the standard TF gRPC server
    * There is one ConfigMap per TfJob; this avoids problems created by a TfJob operator upgrade making changes to the python script.

[33mcommit 383eafd50d81ea2e61b38cb6ac05549a8f20048e[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Oct 18 20:36:23 2017 -0700

    Fix presubmit jobs and periodic jobs (#60)
    
    * Fix presubmit jobs and periodic jobs.
    
    * When testing PR's for other committers we need to fetch the appropriate
      branch.
    
    * For periodic jobs no sha will be set so we need to checkout master
      and then get the sha.

[33mcommit 35fa28c9024fd9ced681742ad878ffbd67300829[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Oct 18 16:28:11 2017 -0700

    PR to test Prow presubmit integration. (#50)
    
    * Add some notes about setting up the k8s bot account to update our PRs with status information.

[33mcommit 0433b203786ab49170fd58446651f6fc3294ccdd[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Oct 17 20:49:01 2017 -0700

    Change bootstrap.py to fix the periodic test. (#56)
    
    * For periodic test REPO_OWNER and REPO_NAME won't be set so we
      need to hard code the value.

[33mcommit 9a97daa51d5dc9df82b918ef6e91a2eec9b1e5d6[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Oct 17 16:20:35 2017 -0700

    E2E test for the CRD (#49)
    
    There are multiple pieces to this test
       * bootstrap.py is a script that runs inside the Docker container started by the ProwJob
          it checks out the repository and then invokes runner.py to run the test.
      * runner.py creates the resources for the test and then invokes helm-test/main.go
     * Create a modified version of helm-test/main.go.
       * Deploys the TfJob helm chart and then runs the tests.
       * Doesn't deploy the chart in its own namespace because the E2E test
         assumes its using default namespace.
    
     * The division of code into runner.py and helm-test/main.go is arbitrary. We should combine them in a future PR into a single go program for the test.
    
     * Add a bunch of needed packages to the container used by the ProwJob.
    
      * changes to helm-test/main.go
           * runner.py Needs to get credentials otherwise helm won't work.
           * Run helm init as part of retry loop waiting for helm to be initialized.
           * Set a 2 minute timeout on waiting for helm to be initialized.
           * Echo command output to logs so that it will show up in job log.
           * Explicitly set the namespace to default; otherwise it looks like we end
    up running in namespace test-pods.
    
     * changes to runner.py
            * Push artifacts to GCS for integration with gubernator
    
     * Store the GCR image path inside the container image and log it
    so we know which image a job ran in.

[33mcommit bfd4de5048873cea846b537062b132b015f04054[m
Author: Sertaç Özercan <sozercan@gmail.com>
Date:   Mon Oct 16 18:43:24 2017 -0700

    updated chart with batch.jobs and extensions.deployments cluster roles (#52)

[33mcommit ce471fde60a379691685dcd853a16e6c1ccda0de[m
Author: Sertaç Özercan <sozercan@gmail.com>
Date:   Mon Oct 16 18:01:04 2017 -0700

    added rbac support for tf-operator chart (#51)

[33mcommit 0f1ffef8639b28b70d8a301b0b16b65651c255e8[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Fri Oct 13 14:35:27 2017 -0700

    Docker image and script for Prow jobs for continuous testing. (#47)
    
    * Create configs for setting up Prow for continuous testing.
    
    * image/... Defines the Docker image used by our ProwJobs.
    * Replace build_and_push.sh with a python script
    * Make registry an argument for build_and_push.py because we will want
      the tests to push to a different registry.
    
     * Add an option to build the Docker image using Google Container Builder
      because Prow won't allow mapping in the Docker socket so we won't be able
      to run Docker in Docker.
    
    * Add README explaining integration with Prow.
    * Add links to the K8s dashboard.

[33mcommit 67e274a1ae00d9dcac43d756ba0524626d59ca98[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Thu Sep 28 22:13:35 2017 -0700

    Fix bug that prevents permanent errors from causing job failure. (#44)
    
    * Fix bug that prevents permanent errors from causing job failure.
    
    * Code should not depend on the termination message being set to decide
      whether to trust exit code. This is legacy code that dependend on a
      script setting the termination message. But the CRD no longer uses
      a script to launch the TF process.

[33mcommit a1738ca980aed62be8c14b24d2ab1e528a1c9bd3[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Wed Sep 27 06:45:22 2017 -0700

    Update the image.

[33mcommit 71c12671e8bf8594fea7d64d8b0b16fc1be589c6[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Sep 27 06:39:12 2017 -0700

    Always check for existing TfJobs and instantiate controllers for them. (#43)
    
    * Always check for existing TfJobs and instantiate controllers for them.
    
    * Whenever we start the controller, we should check if there are existing
      TfJobs and instantiate controllers for them so they aren't orphaned.
    
    * Previously we only did this if the CRD already existed. IMO it seems
      more robust to always do it and not rely on the CRD already existing.
    
    * Added a log message for the watch version obtained.

[33mcommit 78616fc0c11c53f6c1658c243c5aac2785b16645[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Mon Sep 18 18:37:11 2017 -0700

    Update the Docker image to pull in changes to support namespaces.

[33mcommit 8d53edf741fdd3e0b92b057623a17e609e5b8e2f[m
Merge: 46727e52 d2d9c8c9
Author: Jeremy Lewi <jlewi@google.com>
Date:   Mon Sep 18 18:33:45 2017 -0700

    Merge branch 'master' of github.com:jlewi/mlkube.io

[33mcommit 46727e52740bc73330d7cd6aed72752ceea7d6ca[m
Author: loadwiki <chutianqianli@gmail.com>
Date:   Tue Sep 19 09:29:28 2017 +0800

    If tfjob controller runs as a deployment in k8s default namespace, we can't not create tfjob crd in other namespaces.This PR fixes the issue.
    
    There are three major modification:
    
    1）tf job controller should watch crd event in all namespaces.
    
    2）tfjob controller creates/deletes training job/tensorboard in specific namespace instead of its own ns.
    
    3）tfjob controller maintain a map from crd name to crd object.The map key should add crd namespace as prefix.

[33mcommit d2d9c8c9ea1a88620db504b26a5f3fc2d5d28364[m
Author: loadwiki <chutianqianli@gmail.com>
Date:   Tue Sep 19 09:29:28 2017 +0800

    support multi namespaces (#39)

[33mcommit a73fea22b0add7a95b6e246f3c1f42a7c1d455b8[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Sep 13 15:21:52 2017 -0700

        Use Jinja templates and a Python script to build example Docker images for examples (#37)
    
    * Use Jinja templates and a Python script to build example Docker images.
    
    * We use a Jinja template to generate Dockerfiles for CPU and GPU versions
      of the Docker image for the smoke tests. This avoid code duplication from
      maintaining two very similar Dockerfiles.
    
    * We replace the current build_and_push.sh script by a Python script.
      Since the complexity of the script is increasing I prefer to use Python
      to use bash.
    
    * Update the TF Docker images to use TF 1.3 as opposed to RC2.

[33mcommit 69e98b9b844dd7a575e60a98f3ff1518977e5344[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Tue Sep 5 18:45:31 2017 -0700

    Set default values for Replicas, TfPort, TfReplicaType. (#31)
    
    * Default values are set before calling validation so that validation will
      be applied to defaults.
    
    * Picked a random value (2222) as the default port.
    
    * Move creation of TFReplicaSet out of initJob and into TrainingJob.setup.
    
      This is needed because we need to call SetDefaults before creating the TfReplicaSet since SetDefaults sets the default value for number of replicas.
    
    * Update the image used by the chart.

[33mcommit 92e4666784f8fe68c8d1e78efeaf7af9d7599411[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Mon Aug 21 16:42:58 2017 -0700

    Fix a couple bugs. (#27)
    
    * Fix a couple bugs.
    
    * If a job can't be constructed because it fails validation the job should
      be marked as failed and the reason set to an appropriate error.
    
      * We do this by modifying initJob to update the status if there is a problem
        initializing the job.
    
    * replicaStatusFromPodList needs to take into account the state where
      a container is running and there is no lastTerminationState.
    
      * Otherwise we get log spam complaining about no tensorflow container
        found in the running state.
    
    * Clean up some log spam.
    
    * Update the image.

[33mcommit 880d79fe6e3e09f78a97f7015eb842f13ace86d0[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sun Aug 20 13:56:32 2017 -0700

    Use CustomResource instead of ThirdPartyResource. (#20)
    
    * Changes to support CRD.
    
    * Delete WaitForTPR; CRD has new methods for handling the waiting.
    * Create the CRD resource.
    
    * Rename TPR to CRD.
    
    * Remove unused code.
    
    * Tag images with the git commit to ensure unique names.
    * Update the helm test to get the image from a value.
    
    * We need to set Kind and ApiVersion when creating the request otherwise the ApiServer will reject the request with a 400 error.
    
    * Update tpr_util to use the helper methods to construct the URI as opposed to
    constructing the URI manually.
    
    * Add some logging to the E2E test to make it easier to debug.
    
    * Update readme; replace references to TPR with CRD.
    
    * Travis needs to remove some of the vendor'd directories otherwise we get conflicts.

[33mcommit 0bbf48437179fba8203f640ff5f4747e94e0d95a[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Aug 16 11:39:13 2017 -0700

    Update glide config. (#18)
    
    * Remove a bunch of unused dependencies (aws, coreos, etc...)
    * Rename the package to match the full name; i.e. github.com/jlewi/mlkube.io

[33mcommit 7fe33077755468faa4efe190d2462b3417759ae3[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Wed Aug 16 11:00:29 2017 -0700

    * Update examples to use a specific TF image as opposed to latest.
    * We use the forthcoming RC for TF 1.3.
    * Change references to cmle to mlkube.

[33mcommit 392d54c6c811cdf6f47ec2ea75a5185369f38e78[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Tue Aug 15 21:06:55 2017 -0400

    Add TensorBoard Integration (#15)
    
    * Add TensorBoard
    
    * Add tests for TensorBoard + review comments
    
    * Update doc and example
    
    * fix glide issue on travis

[33mcommit 3babc92c158e58dae0f1109acd0864a5f856f510[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Sat Aug 5 16:00:28 2017 -0700

    Travis widget should link to the latest build.

[33mcommit fc893fe0cbb5409bf8c48cc8163e6a62e22efcae[m
Merge: 033ad2d8 7d8d8775
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sat Aug 5 15:51:31 2017 -0700

    Merge pull request #14 from jlewi/travis_test
    
    Changes to support CI using Travis.

[33mcommit 7d8d87750c477a229ca083b2b4be7ac79a7eb8e3[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Sat Aug 5 12:44:55 2017 -0700

    Make travis work.
    
    * Configure Travis to use glide to pull in dependencies.
    * Override the default Travis test command to exclude tests in the Vendor
      directory.
    
    * Renamed the mlkube imports to be github.com/jlewi/mlkube.io
      * I think this is needed because Travis will checkout the code as
       gopath/src/github.com/jlewi/mlkube.io
      * So changing the import is needed to make it work with Travis.
      * It also appears to be more consistent with gostyle and compatible with go deps.
    
    * Add Travis widget to the README.
    
    * Fix the readme and build scripts.

[33mcommit 033ad2d818254b071223aca2119299c0167f2fe1[m
Merge: cf696435 dfb66dbd
Author: Jeremy Lewi <jlewi@google.com>
Date:   Sat Aug 5 12:40:28 2017 -0700

    Merge branch 'master' of github.com:jlewi/mlkube.io

[33mcommit cf69643563e53d80c225557342821d84c9a4333c[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Sat Aug 5 12:39:02 2017 -0700

    Add a Travis file.

[33mcommit dfb66dbdaa67900bb9d1f120e4cc118e5ee69e7a[m
Merge: 452a8db9 eb59fc6a
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Aug 2 12:17:13 2017 -0700

    Merge pull request #12 from wbuchwalter/config-env
    
    Add Environment Variables in Controller Config

[33mcommit 452a8db9c42fb5cb6e40a2da57fe351fc1d367b9[m
Merge: dcc984ce c9e5efb3
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Aug 2 12:14:53 2017 -0700

    Merge pull request #10 from wbuchwalter/rename-chart
    
    Helm charts renaming

[33mcommit dcc984ce1ce9d3a0686ab756e8b7d81358f0c165[m
Merge: dbc80095 01bafa0e
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Aug 2 08:40:59 2017 -0700

    Merge pull request #11 from wbuchwalter/fix-tests
    
    Fix tests

[33mcommit eb59fc6a434409317499ba37adaee51163d5798b[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Wed Aug 2 11:35:16 2017 -0400

    cleanup

[33mcommit c97f4655e6bc04b5ac14e40b6fbe4cfe2ffada91[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Wed Aug 2 11:28:39 2017 -0400

    remove singlenode example

[33mcommit fe1e10b54f5103abd6b9e8b0c93862078297dc39[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Wed Aug 2 10:47:52 2017 -0400

    Add EnvVars in controller config

[33mcommit c9e5efb33c7bd710fc9ea70624d247bad1d9d5a0[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Tue Aug 1 17:30:26 2017 -0400

    rename example chart

[33mcommit 01bafa0e440ddbb589531c7963cfeb138e1d92e1[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Tue Aug 1 17:29:10 2017 -0400

    fix tests

[33mcommit 4dd85ddf764c9b867b1916703a11dda104e0f2c2[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Tue Aug 1 11:32:16 2017 -0400

    rename chart to tf-job-operator

[33mcommit dbc80095f7459f5d3d481611cfa890f4cf6fc7eb[m
Merge: 4bd9c089 fd494a82
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Mon Jul 31 21:24:28 2017 -0700

    Merge pull request #9 from jlewi/gpus
    
    Simplify GPU configuration process.

[33mcommit fd494a826049df627a6fa4903e955c89b3252441[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Mon Jul 31 10:05:24 2017 -0700

    Simplify GPU configuration process.
    
    * Job controller is configured with a list of volumes that need to be added to pods that use GPUs.
    
    * Controller looks at pods that specify GPU resources and adds volume mounts needed for GPUs.

[33mcommit 4bd9c089e24677117e03ec0e1f68a5b13068617c[m
Merge: 1184c3d4 09eaeb24
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Mon Jul 31 06:48:44 2017 -0700

    Merge pull request #8 from wbuchwalter/fix-dependencies
    
    Fix build, add Glide for dependency management.

[33mcommit 09eaeb24f7c10c8b44f6b007d2819bbf96850087[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Mon Jul 31 09:43:13 2017 -0400

    Update install doc

[33mcommit 2698fe4d42c28db303db2f06e756cdac10cf26b9[m
Merge: 978a152e 1184c3d4
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Mon Jul 31 09:33:17 2017 -0400

    Merge branch 'master' into fix-dependencies

[33mcommit 978a152edab29e19f8668b12cfec662224fac2ff[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Mon Jul 31 09:26:35 2017 -0400

    run gofmt

[33mcommit 1184c3d45af7361106c455c58ca555504bc2622d[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Sun Jul 30 21:02:16 2017 -0700

    Reformat all files using gofmt.
    
    find ./ -name *.go -exec gofmt -w {} ";"

[33mcommit abd44b6960163290e0dadc0aa053cdebdb0a991c[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Sun Jul 30 20:26:21 2017 -0400

    fix indentation

[33mcommit b0fa9f8a4dcb48d745345060eebd00e352db2ca3[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Sun Jul 30 20:16:34 2017 -0400

    undo examples changes

[33mcommit 247b9fe4423d84343f68c9c73617d0d223e83d31[m
Author: William Buchwalter <will@Williams-MacBook-Pro.local>
Date:   Fri Jul 28 14:10:44 2017 -0700

    fix dependencies

[33mcommit 7688fe64de43cc218345fba1219092c831ca8f73[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Tue Jul 25 22:20:02 2017 -0700

    Fix a legacy reference to the cmle.io package; it should be mlkube.io.

[33mcommit 52342c86d78eb4eef00bef5ec667d8165f2f20a0[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Tue Jul 25 19:46:13 2017 -0700

    Fix a Print statement.

[33mcommit 052e1783c7e21d5b761820491d548e71c108bd76[m
Merge: d0d66440 d398958c
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sun Jul 23 16:47:35 2017 -0700

    Merge pull request #2 from jlewi/testing
    
    A more thorough E2E test.

[33mcommit d0d664401b3489d8c0ea28a3cf5a039b2eadcb52[m
Merge: b9225ae9 3e06afe4
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Sun Jul 23 16:43:47 2017 -0700

    Merge pull request #3 from wbuchwalter/patch-1
    
    Update links in README.md

[33mcommit 3e06afe4b2f2f6c1c0de63ad8d666269f80616ac[m
Author: William Buchwalter <wbuchwalter@gmail.com>
Date:   Thu Jul 20 18:07:46 2017 -0400

    Update links in README.md

[33mcommit d398958cfccb0276ff929371b6a70f2eaac715b2[m
Merge: 2f8ceb3a 2d16c9a9
Author: Jeremy Lewi <jlewi@google.com>
Date:   Tue Jul 18 07:39:07 2017 -0700

    Merge remote-tracking branch 'github/testing' into testing

[33mcommit 2f8ceb3a9281c884f6d798aae3c71d0ed0d2cb80[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Thu Jul 13 07:03:13 2017 -0700

    A more thorough E2E test.
    
    * Rewrite the helm test as a go program (as opposed to Python).
    * Bake the go program into the Docker image for the operator; as opposed to using a config map.
    * The go program allows us to write more detailed tests such as verifying that the resources spawned by the TfJob are created and deleted.
    * The test is still pretty minimal.
    * Support TAP output.

[33mcommit 2d16c9a91b6443a19273c8590170833be6465290[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Tue Jul 18 07:22:22 2017 -0700

    A more thorough E2E test.
    
    * Rewrite the helm test as a go program (as opposed to Python).
    * Bake the go program into the Docker image for the operator; as opposed to using a config map.
    * The go program allows us to write more detailed tests such as verifying that the resources spawned by the TfJob are created and deleted.
    * The test is still pretty minimal.

[33mcommit b9225ae957bd383c7b02aa29b495aa1741f9b773[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Thu Jul 13 07:03:13 2017 -0700

    Fix tensorboard chart.
    
    * logDir not properly templated.
    * We should use a deployment so that pods can be updated.

[33mcommit b922932b96261bac2fbcb51f30b6bdd7ec1db0e8[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Thu Jul 13 06:45:16 2017 -0700

    Add a helm chart for the example.
    
    We leave the tf_job.yaml file for now because it is used by the E2E tests.

[33mcommit 80653e45acc01ec0afd476e1173189e2a0efd750[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Thu Jul 13 06:09:47 2017 -0700

    Set RuntimeId automatically to a random/unique id.
    
    * RuntimeId is used to ensure resources created by the TfJob have unique names.
    * The user shouldn't have to specify this. Eventually, we will probably
      remove it from the spec altogether and maybe make it metadata like the
      UID in JobController.
    * Define a TfJobClient interface for managing TfJobs. The interface facilitates
      testing because we can use a Fake.

[33mcommit ca1b40545089356925c91a46796275978af27f89[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Wed Jul 12 21:14:04 2017 -0700

    Create a helm chart for Running TensorBoard.

[33mcommit 3fea4404209b3ac61f59d5596d16ca30c7974463[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Tue Jul 11 20:50:46 2017 -0700

    Deploy the TfJob operator using a helm chart.
      * The chart includes a basic test to make sure the operator is working.

[33mcommit d235a030ea1791e5a65d4ef79c95f7204480b9b5[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Fri Jul 7 05:24:49 2017 -0700

    Add some thoughts about potential integration of TensorBoard.

[33mcommit cfd24b1ea317a3dbd9fb60f622c05cf23af3c924[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Thu Jul 6 14:11:50 2017 -0700

    Rename waitCluster -> waitJobs.

[33mcommit 20f5f9c20a9f7196b9f073c474d557d2cb930294[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Thu Jul 6 14:09:33 2017 -0700

    Fix training_test.

[33mcommit 437454fe0273a827b630c76a14699f17ff4aea34[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Thu Jul 6 09:39:38 2017 -0700

    Fix replicas_test; the test now passes.
    
    * Comment out training_test since that's not working yet.

[33mcommit c243a805c3a5661495b059310572e510be5acba9[m
Author: Jeremy Lewi <jlewi@google.com>
Date:   Wed Jun 28 11:40:57 2017 -0700

    Initial commit of a K8s TPR and operator for TensorFlow training jobs.

[33mcommit 5b1ff9c7058c2af718ed8d399aebcfd124217f8c[m
Author: Jeremy Lewi <jeremy@lewi.us>
Date:   Wed Jun 28 11:38:15 2017 -0700

    Initial commit
