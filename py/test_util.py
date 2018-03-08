import errno
# TODO(jlewi): Callers should be using the version of This
# file in kubeflow/testing. We should delete this file.
import logging
import os
import subprocess
import time
from xml.etree import ElementTree

import six

from py import util


class TestCase(object):

  def __init__(self, class_name="", name=""):
    self.class_name = class_name
    self.name = name
    # Time in seconds of the test.
    self.time = None
    # String describing the failure.
    self.failure = None


class TestSuite(object):
  """A suite of test cases."""

  def __init__(self, class_name):
    self._cases = {}
    self._class_name = class_name

  def create(self, name):
    """Create a new TestCase with the specified name.

    Args:
      name: Name for the newly created TestCase.

    Returns:
      TestCase: The newly created test case.

    Raises:
      ValueError: If a test case with the specified name already exists.
    """
    if name in self._cases:
      raise ValueError("TestSuite already has a test named %s" % name)
    self._cases[name] = TestCase()
    self._cases[name].class_name = self._class_name
    self._cases[name].name = name
    return self._cases[name]

  def get(self, name):
    """Get the specified test case.

    Args:
      name: Name of the test case to return.

    Returns:
      TestCase: The requested test case.

    Raises:
      KeyError: If no test with that name exists.
    """
    if not name in self._cases:
      raise KeyError("No TestCase named %s" % name)
    return self._cases[name]

  def __iter__(self):
    """Return an iterator of TestCases."""
    return six.itervalues(self._cases)


def wrap_test(test_func, test_case):
  """Wrap a test func.

  Test_func is a callable that contains the commands to perform a particular
  test.

  Args:
    test_func: The callable to invoke.
    test_case: A TestCase to be populated.

  Raises:
    Exceptions are reraised to indicate test failure.
  """
  start = time.time()
  try:
    test_func()
  except subprocess.CalledProcessError as e:
    test_case.failure = ("Subprocess failed;\n{0}".format(e.output))
    raise
  except Exception as e:
    test_case.failure = "Test failed; " + e.message
    raise
  finally:
    test_case.time = time.time() - start


def create_xml(test_cases):
  """Create an Element tree representing the test cases.

  Args:
    test_cases: TestSuite or List of test case objects.

  Returns:
    ElementTree: representing the elements.
  """
  total_time = 0
  failures = 0
  for c in test_cases:
    if c.time:
      total_time += c.time

    if c.failure:
      failures += 1
  attrib = {
    "failures": "{0}".format(failures),
    "tests": "{0}".format(len(test_cases)),
    "time": "{0}".format(total_time)
  }
  root = ElementTree.Element("testsuite", attrib)

  for c in test_cases:
    attrib = {
      "classname": c.class_name,
      "name": c.name,
    }
    if c.time:
      attrib["time"] = "{0}".format(c.time)

    # If the time isn't set and no message is set we interpret that as
    # the test not being run.
    if not c.time and not c.failure:
      c.failure = "Test was not run."

    e = ElementTree.Element("testcase", attrib)

    root.append(e)

    if c.failure:
      f = ElementTree.Element("failure")
      f.text = c.failure
      e.append(f)

  t = ElementTree.ElementTree(root)
  return t


def create_junit_xml_file(test_cases, output_path, gcs_client=None):
  """Create a JUnit XML file.

  The junit schema is specified here:
  https://www.ibm.com/support/knowledgecenter/en/SSQ2R2_9.5.0/com.ibm.rsar.analysis.codereview.cobol.doc/topics/cac_useresults_junit.html

  Args:
    test_cases: TestSuite or List of test case objects.
    output_path: Path to write the XML
    gcs_client: GCS client to use if output is GCS.
  """
  t = create_xml(test_cases)
  logging.info("Creating %s", output_path)
  if output_path.startswith("gs://"):
    b = six.StringIO()
    t.write(b)

    bucket_name, path = util.split_gcs_uri(output_path)
    bucket = gcs_client.get_bucket(bucket_name)
    blob = bucket.blob(path)
    blob.upload_from_string(b.getvalue())
  else:
    dir_name = os.path.dirname(output_path)
    if not os.path.exists(dir_name):
      logging.info("Creating directory %s", dir_name)
      try:
        os.makedirs(dir_name)
      except OSError as e:
        if e.errno == errno.EEXIST:
          # The path already exists. This is probably a race condition
          # with some other test creating the directory.
          # We should just be able to continue
          pass
        else:
          raise
    t.write(output_path)


def get_num_failures(xml_string):
  """Return the number of failures based on the XML string."""

  e = ElementTree.fromstring(xml_string)
  return int(e.attrib.get("failures", 0))
