"""Test console recovery with a fake binary that replaces itself and fails."""
from pathlib import Path
import subprocess
import tempfile
import unittest

class WrapperTest(unittest.TestCase):
    def test_replaced_binary_still_restores_and_reports_error(self):
        with tempfile.TemporaryDirectory(prefix="mz-wrapper-") as tmp:
            root=Path(tmp)
            binary=root/"misterzine"
            binary.write_text('#!/bin/bash\nif [ "$1" = console-restore ]; then echo restored >> "'+tmp+'/restored"; exit 0; fi\nprintf "#!/bin/bash\\nexit 99\\n" > "'+tmp+'/replacement"\nmv "'+tmp+'/replacement" "'+tmp+'/misterzine"\nexit 7\n')
            binary.chmod(0o700)
            (root/"log.txt").write_text("framebuffer unavailable fixture\n")
            source=(Path(__file__).parent.parent/"deploy/launch.sh").read_text()
            source=source.replace("DIR=/media/fat/misterzine",'DIR="'+tmp+'"')
            source=source.replace("/tmp/misterzine-restore.XXXXXX",tmp+"/restore.XXXXXX")
            wrapper=root/"wrapper.sh";wrapper.write_text(source)
            result=subprocess.run(["bash",str(wrapper)],input="\n",text=True,capture_output=True,timeout=5)
            self.assertEqual(result.returncode,7,result.stderr)
            self.assertIn("framebuffer unavailable fixture",result.stdout)
            self.assertIn("status 7",result.stdout)
            self.assertEqual((root/"restored").read_text().splitlines(),["restored","restored"])
            self.assertEqual(list(root.glob("restore.*")),[])

if __name__=="__main__": unittest.main()
