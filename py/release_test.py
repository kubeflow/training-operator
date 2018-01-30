import tempfile
import unittest

import mock
import yaml

from py import release


class ReleaseTest(unittest.TestCase):
  @mock.patch("py.release.os.makedirs")
  @mock.patch("py.release.os.symlink")
  @mock.patch("py.release.util.install_go_deps")
  @mock.patch("py.release.util.clone_repo")
  @mock.patch("py.release.build_and_push")
  def test_build_postsubmit(self, mock_build_and_push, mock_clone,    # pylint: disable=no-self-use
                            _mock_install, _mock_os, _mock_makedirs):
    parser = release.build_parser()
    args = parser.parse_args(["postsubmit", "--src_dir=/top/src_dir"])
    release.build_postsubmit(args)

    mock_build_and_push.assert_called_once_with(
      '/top/src_dir/go', '/top/src_dir/go/src/github.com/tensorflow/k8s',
      mock.ANY)
    mock_clone.assert_called_once_with(
      '/top/src_dir/git_tensorflow_k8s', 'tensorflow', 'k8s', None, None)

  @mock.patch("py.release.os.makedirs")
  @mock.patch("py.release.os.symlink")
  @mock.patch("py.release.util.install_go_deps")
  @mock.patch("py.release.util.clone_repo")
  @mock.patch("py.release.build_and_push")
  def test_build_pr(self, mock_build_and_push, mock_clone, _mock_install, _mock_os, _mock_makedirs):  # pylint: disable=no-self-use
    parser = release.build_parser()
    args = parser.parse_args(["pr", "--pr=10", "--commit=22",
                              "--src_dir=/top/src_dir"])
    release.build_pr(args)

    mock_build_and_push.assert_called_once_with(
      '/top/src_dir/go', '/top/src_dir/go/src/github.com/tensorflow/k8s',
      mock.ANY)
    mock_clone.assert_called_once_with(
      "/top/src_dir/git_tensorflow_k8s", "tensorflow", "k8s", "22",
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
      self.assertEquals(expected, output)

  def test_update_chart_file(self):
    with tempfile.NamedTemporaryFile(delete=False) as hf:
      hf.write("""
name: tf-job-operator-chart
home: https://github.com/tensorflow/k8s
version: 0.1.0
appVersion: 0.1.0
""")
      chart_file = hf.name

    release.update_chart(chart_file, "v20171019")

    with open(chart_file) as hf:
      output = yaml.load(hf)
    expected = {
        "name": "tf-job-operator-chart",
        "home": "https://github.com/tensorflow/k8s",
        "version": "0.1.0-v20171019",
        "appVersion": "0.1.0-v20171019",
    }
    self.assertEquals(expected, output)


if __name__ == "__main__":
  unittest.main()
