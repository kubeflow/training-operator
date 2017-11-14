from __future__ import print_function

import tempfile
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
                """<testcase classname="some_test" """
                """failure="failed for some reason." name="first" """
                """time="10" /></testsuite>""")

    self.assertEquals(expected, output)


if __name__ == "__main__":
  unittest.main()
