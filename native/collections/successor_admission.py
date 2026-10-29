"""Existing admission format2 with explicit authority1/2 reader support."""
import runtime_authority as original
import shared_admission as shared
import successor_authority as authority
import journal as j

FORMAT = 2

class Gate(shared.Gate):
    def _check_authority(self,row,*,shell=False):
        opened=authority.open_authority(row['authority']['root'])
        try:
            opened.verify_reservation(row['authority'],self.root,row['permit'])
            if shell:
                display=shared.display_for_shell(row['permit']['shell'])
                j.require(str(display)==opened.descriptor['context']['socket']['path'] and display.parent/original.ANCHOR==opened.root,
                          'permit authority is not the shell display authority')
        finally:opened.close()
