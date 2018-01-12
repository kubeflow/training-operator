from __future__ import print_function

import subprocess
import tempfile
import time
import unittest

from py import test_util

class XMLTest(unittest.TestCase):
  def test_write_xml(self):
    with tempfile.NamedTemporaryFile(delete=False) as hf:
      pass

    success = test_util.TestCase()
    success.class_name = "some_test"
    success.name = "first"
    success.time = 10

    failure = test_util.TestCase()
    failure.class_name = "some_test"
    failure.name = "first"
    failure.time = 10
    failure.failure = "failed for some reason."

    test_util.create_junit_xml_file([success, failure], hf.name)
    with open(hf.name) as hf:
      output = hf.read()
      print(output)
    expected = ("""<testsuite failures="1" tests="2" time="20">"""
                """<testcase classname="some_test" name="first" time="10" />"""
                """<testcase classname="some_test" name="first" """
                """time="10"><failure>failed for some reason.</failure>"""
                """</testcase></testsuite>""")

    self.assertEquals(expected, output)


class TestSuiteTest(unittest.TestCase):
  def testSuite(self):
    """Test TestSuite."""
    s = test_util.TestSuite("test_class")
    c1 = s.create("c1")
    c1.time = 100

    c2 = s.create("c2")
    c2.time = 200

    c1_get = s.get("c1")
    self.assertEquals(100, c1_get.time)

    c2_get = s.get("c2")
    self.assertEquals(200, c2_get.time)

    names = set()

    for c in s:
      names.add(c.name)

    self.assertItemsEqual(["c1", "c2"], names)

class TestWrapTest(unittest.TestCase):
  def testOk(self):
    def ok():
      time.sleep(1)
      pass

    t = test_util.TestCase()
    test_util.wrap_test(ok, t)
    self.assertGreater(t.time, 0)
    self.assertEquals(None, t.failure)

  def testSubprocessError(self):
    def run():
      raise subprocess.CalledProcessError(10, "some command", output="some output")

    t = test_util.TestCase()
    self.assertRaises(subprocess.CalledProcessError, test_util.wrap_test, run, t)
    self.assertGreater(t.time, 0)
    self.assertEquals("Subprocess failed;\nsome output", t.failure)

  def testGeneralError(self):
    def run():
      time.sleep(1)
      raise ValueError("some error")

    t = test_util.TestCase()
    self.assertRaises(ValueError, test_util.wrap_test, run, t)
    self.assertGreater(t.time, 0)
    self.assertEquals("Test failed; some error", t.failure)

if __name__ == "__main__":
  unittest.main()
