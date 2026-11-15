#!/usr/bin/env python3
"""Validate mounting and passenger UI against disposable GX login/world services."""
import os,secrets,shutil,socket,sqlite3,subprocess,tempfile,time
from pathlib import Path
from update_mount_content import patch,RACES
import configparser

root=Path(__file__).resolve().parents[1]
children=[]
with tempfile.TemporaryDirectory(prefix='gx-mounts-n2-') as temp:
 temp=Path(temp);database=temp/'rapture.sqlite'
 with sqlite3.connect(root/'server/data/rapture.sqlite') as src,sqlite3.connect(database) as dst:src.backup(dst)
 env=dict(os.environ,RAPTURE_DB_PATH=str(database),RAPTURE_DATA_DIR=str(root/'server/data'),RAPTURE_LOGIN_LISTEN='127.0.0.1:40820',RAPTURE_WORLD_LISTEN='127.0.0.1:40830',RAPTURE_PUBLIC_IP='127.0.0.1',SHAIYA_TEST_LOGIN_ADDR='127.0.0.1:40820',SHAIYA_TEST_WORLD_PORT='40830',SHAIYA_TEST_DISPOSABLE='1',SHAIYA_TEST_RENDER='1',SHAIYA_TEST_VEHICLE='1',SHAIYA_TEST_ARTIFACTS=str(root/'work/mount-system'),SHAIYA_DATA=os.environ.get('SHAIYA_DATA',str(Path.home()/'Desktop/Shaiya GX/data')),GOTOOLCHAIN='local')
 env['PATH']=str(Path.home()/'.local/opt/go-1.27.3/bin')+':'+env['PATH']
 for port in (40820,40830):
  with socket.socket() as check:
   if check.connect_ex(('127.0.0.1',port))==0:raise RuntimeError(f'Test port {port} already in use')
 users=[]
 with sqlite3.connect(database) as db:
  db.row_factory=sqlite3.Row
  bindings={}
  for race in RACES.values():
   cfg=configparser.ConfigParser();cfg.read(root/'resources/vehicle'/race/'bones.ini');bindings[race]=cfg
  _,mounts=patch((Path(env['SHAIYA_DATA'])/'Item/Item.SData').read_bytes(),bindings)
  db.executemany('UPDATE PS_GameDefs_Items SET Range=?,AttackTime=?,Speed=? WHERE Type=42 AND TypeID=?',[(*meta,i) for i,meta in mounts.items()])
  template=dict(db.execute('SELECT * FROM PS_GameData_Chars WHERE Del=0 LIMIT 1').fetchone())
  for n in range(2):
   user='mount'+secrets.token_hex(5);password=secrets.token_hex(6)
   uid=db.execute('SELECT MAX(UserUID)+1 FROM PS_UserData_Users_Master').fetchone()[0]
   charid=db.execute('SELECT MAX(CharID)+1 FROM PS_GameData_Chars').fetchone()[0]
   db.execute("INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(?,?,?,0,0,0,0,0,'N',0)",(uid,user,password))
   db.execute('INSERT INTO PS_GameData_UserMaxGrow(ServerID,UserUID,Country,MaxGrow,Del) VALUES(1,?,0,3,0)',(uid,))
   ch=dict(template);ch.update(UserID=user,UserUID=uid,CharID=charid,CharName='Mt'+secrets.token_hex(4),Slot=0,Family=0,Job=0,Sex=0,Grow=2,Level=60,HP=50000,Map=1,PosX=578+n*2,PosY=78,Posz=1762,Del=0)
   db.execute('INSERT INTO PS_GameData_Chars('+','.join('['+k+']' for k in ch)+') VALUES('+','.join('?' for _ in ch)+')',list(ch.values()))
   if n==0:
    itemuid=db.execute('SELECT MAX(ItemUID)+1 FROM PS_GameData_CharItems').fetchone()[0]
    db.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,42005,?,42,5,0,13,0,0,0,0,0,0,0,'',1,datetime('now'),'N2',0)",(charid,itemuid))
   users.append((user,password))
 env.update(SHAIYA_TEST_USER=users[0][0],SHAIYA_PASSWORD=users[0][1],SHAIYA_TEST_PARTNER=users[1][0],SHAIYA_TEST_PARTNER_PASSWORD=users[1][1])
 try:
  for name,port in [('gx_login',40820),('gx_world',40830)]:
   log=open(root/('work/mount-system/'+name+'.log'),'w')
   child=subprocess.Popen([str(root/'server/build'/name)],cwd=temp,env=env,stdout=log,stderr=subprocess.STDOUT);children.append(child)
   deadline=time.monotonic()+30
   while True:
    with socket.socket() as s:
     if s.connect_ex(('127.0.0.1',port))==0:break
    if child.poll() is not None or time.monotonic()>deadline:raise RuntimeError('Service startup failed: '+name)
    time.sleep(.1)
  subprocess.run(['go','test','-tags','integration','./internal/app','-run','^TestVehicle(AgainstRapture|ModelsRealGL)$','-count=1','-v'],cwd=root,env=env,check=True,timeout=120)
 finally:
  for child in reversed(children):
   child.terminate()
   try:child.wait(timeout=10)
   except subprocess.TimeoutExpired:child.kill();child.wait()
 print('Mount N2: disposable services and accounts removed.')
