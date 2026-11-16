#!/usr/bin/env python3
"""Initialize/query the portable development database or create a local account."""
import argparse,os,shutil,sqlite3,subprocess
from pathlib import Path
from runtime_lock import runtime_lock

ROOT=Path(__file__).resolve().parents[1]

def main():
 p=argparse.ArgumentParser(description=__doc__)
 p.add_argument('--path',type=Path,default=Path(os.environ.get('RAPTURE_DB_PATH',ROOT/'runtime/rapture.sqlite')))
 sub=p.add_subparsers(dest='action',required=True)
 sub.add_parser('init')
 q=sub.add_parser('query');q.add_argument('--file',type=Path,required=True)
 a=sub.add_parser('account');a.add_argument('name');a.add_argument('--admin',action='store_true')
 args=p.parse_args();path=args.path.expanduser().resolve()
 if args.action=='account':
  executable=ROOT/'build'/('gx_console.exe' if os.name=='nt' else 'gx_console')
  if not executable.is_file():raise RuntimeError('build gx_console first; account creation uses the shared Argon2id implementation')
  environment=dict(os.environ,RAPTURE_DB_PATH=str(path),RAPTURE_DB_SEED=str(ROOT/'data/rapture.sqlite'))
  command=[str(executable),'account','create',args.name]
  if args.admin:command.append('--admin')
  subprocess.run(command,env=environment,check=True)
  return
 path.parent.mkdir(parents=True,exist_ok=True)
 if not path.exists():
  with runtime_lock(path):
   if not path.exists():
    temp=path.with_name('.'+path.name+'.next')
    shutil.copyfile(ROOT/'data/rapture.sqlite',temp)
    temp.chmod(0o600)
    temp.replace(path)
 with sqlite3.connect(path) as c:
  c.execute('PRAGMA busy_timeout=10000')
  if c.execute('PRAGMA user_version').fetchone()[0]!=1:raise RuntimeError('unsupported schema')
  if args.action=='init':print('SQLite ready:',path)
  elif args.action=='query':
   # Queries use explicit PS_GameData_/PS_UserData_/PS_GameDefs_ table prefixes.
   statement=''
   statements=[]
   for char in args.file.read_text():
    statement+=char
    if char==';' and sqlite3.complete_statement(statement):
     statements.append(statement);statement=''
   if statement.strip():statements.append(statement)
   for statement in statements:
    result=c.execute(statement)
    if result.description:
     for row in result:print('\t'.join('NULL' if x is None else str(x) for x in row))

if __name__=='__main__':main()
