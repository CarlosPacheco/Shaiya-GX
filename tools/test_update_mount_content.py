import struct
import unittest
from update_mount_content import patch,load_bindings
from update_recreation_content import string

class MountMigrationTest(unittest.TestCase):
    def test_preserves_native_bytes_and_cooldown(self):
        bindings=load_bindings()
        records=[]
        # Native low/high horses, level-40 variant, double-seat special, board, reindeer.
        for ident,level,grow,model in [(1,25,1,0),(5,50,2,0),(9,40,1,1),(17,50,2,2),(25,28,1,5),(29,28,1,6)]:
            body=bytearray(range(81));body[0]=42;body[1]=ident;body[6]=0;body[13]=grow;body[31]=20;body[32]=model;body[33]=5;body[65]=0
            struct.pack_into('<H',body,4,level)
            records.append(string(f'Mount {ident}')+string('Original description')+body)
        raw=struct.pack('<II',1,len(records))+b''.join(records)
        output,rows=patch(raw,bindings)
        self.assertEqual(rows,{1:(0,1,4),5:(1,2,5),9:(2,1,4),17:(4,2,5),25:(6,1,4),29:(7,1,4)})
        self.assertEqual(len(raw),len(output))
        differences={i for i,(a,b) in enumerate(zip(raw,output)) if a!=b}
        start=8
        allowed=set()
        for record in records:
            body_start=start+len(record)-81
            allowed.update(body_start+i for i in (32,33,65));start+=len(record)
        self.assertLessEqual(differences,allowed)
        again,second=patch(output,bindings)
        self.assertEqual(again,output);self.assertEqual(second,rows)
