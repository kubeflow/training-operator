import tempfile
import unittest

import mock

from py import release


class ReleaseTest(unittest.TestCase):

  @mock.patch("py.release.os.makedirs")
  @mock.patch("py.release.os.symlink")
  @mock.patch("py.release.util.install_go_deps")
  @mock.patch("py.release.util.clone_repo")
  @mock.patch("py.release.build_and_push")
  def test_build_postsubmit(  # pylint: disable=no-self-use
      self, mock_build_and_push, mock_clone, _mock_install, _mock_os,
      _mock_makedirs):
    # Make sure REPO_OWNER and REPO_NAME aren't changed by the environment
    release.REPO_ORG = "kubeflow"
    release.REPO_NAME = "tf-operator"

    parser = release.build_parser()
    args = parser.parse_args(["postsubmit", "--src_dir=/top/src_dir"])
    release.build_postsubmit(args)

    mock_build_and_push.assert_called_once_with(
      '/top/src_dir/go', '/top/src_dir/go/src/github.com/kubeflow/tf-operator',
      mock.ANY)
    mock_clone.assert_called_once_with('/top/src_dir/git_tensorflow_k8s',
                                       'kubeflow', 'tf-operator', None, None)

  @mock.patch("py.release.os.makedirs")
  @mock.patch("py.release.os.symlink")
  @mock.patch("py.release.util.install_go_deps")
  @mock.patch("py.release.util.clone_repo")
  @mock.patch("py.release.build_and_push")
  def test_build_pr(# pylint: disable=no-self-use
      self,
      mock_build_and_push,
      mock_clone,
      _mock_install,
      _mock_os,
      _mock_makedirs):
    parser = release.build_parser()
    args = parser.parse_args(
      ["pr", "--pr=10", "--commit=22", "--src_dir=/top/src_dir"])
    release.build_pr(args)

    mock_build_and_push.assert_called_once_with(
      '/top/src_dir/go', '/top/src_dir/go/src/github.com/kubeflow/tf-operator',
      mock.ANY)
    mock_clone.assert_called_once_with("/top/src_dir/git_tensorflow_k8s",
                                       "kubeflow", "tf-operator", "22",
                                       ["pull/10/head:pr"])

  def test_update_values(self):
    with tempfile.NamedTemporaryFile(delete=False) as hf:
      hf.write("""# Test file
image: gcr.io/image:latest

## Install Default RBAC roles and bindings
rbac:
  install: false
  apiVersion: v1beta1""")
      values_file = hf.name

    release.update_values(hf.name, "gcr.io/image:v20171019")

    with open(values_file) as hf:
      output = hf.read()

      expected = """# Test file
image: gcr.io/image:v20171019

## Install Default RBAC roles and bindings
rbac:
  install: false
  apiVersion: v1beta1"""
      self.assertEqual(expected, output)

if __name__ == "__main__":
  unittest.main()
