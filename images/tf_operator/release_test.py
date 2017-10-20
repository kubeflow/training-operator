from images.tf_operator import release
import tempfile
import unittest

class ReleaseTest(unittest.TestCase):
  def test_update_values(self):
    with tempfile.NamedTemporaryFile(delete=False) as hf:
      hf.write("""# Test file
image: gcr.io/image:latest

## Install Default RBAC roles and bindings
rbac:
  install: false
  apiVersion: v1beta1""")
      values_file = hf.name

  release.update_values(hf.name, "gcr.image:v20171019")

  with open(values_file) as hf:
    output = hf.read()

    expected = """# Test file
image: gcr.io/image:v20171019

## Install Default RBAC roles and bindings
rbac:
  install: false
  apiVersion: v1beta1"""
    self.assertEquals(expected, output)

if __name__ == "__main__":
  unittest.main()