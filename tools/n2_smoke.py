#!/usr/bin/env python3
"""N2 against isolated SQLite login/world processes and a disposable DB copy.

Requires Rapture's build/ binaries and the original client assets. No Docker,
SQL credentials, personal runtime DB or existing game service is used.
"""
import os,re,secrets,shutil,socket,sqlite3,subprocess,tempfile,time
from pathlib import Path
client=Path(__file__).resolve().parents[1]
server=Path(os.environ.get('GX_SERVER_DIR',os.environ.get('RAPTURE_DIR',client/'server'))).expanduser().resolve()
children=[]
with tempfile.TemporaryDirectory(prefix='rapture-sqlite-n2-') as temp:
 temp=Path(temp);db=temp/'rapture.sqlite';shutil.copyfile(server/'data/rapture.sqlite',db)
 env=dict(os.environ,RAPTURE_DB_PATH=str(db),RAPTURE_DATA_DIR=str(server/'data'),RAPTURE_LOGIN_LISTEN='127.0.0.1:40800',RAPTURE_WORLD_LISTEN='127.0.0.1:40810',RAPTURE_PUBLIC_IP='127.0.0.1',SHAIYA_TEST_LOGIN_ADDR='127.0.0.1:40800',SHAIYA_TEST_WORLD_PORT='40810',SHAIYA_TEST_DISPOSABLE='1',SHAIYA_TEST_RENDER='1',SHAIYA_TEST_NATIVE_CONTROLS='1',SHAIYA_TEST_ACTIONS='1',SHAIYA_TEST_ARTIFACTS=str(client/'work'),SHAIYA_DATA=os.environ.get('SHAIYA_DATA',str(Path.home()/'Desktop/Shaiya GX/data')),GOTOOLCHAIN='local')
 data=temp/'data';data.mkdir()
 shutil.copytree(server/'data/game',data/'game')
 for name in ('maps','geometry'):
  (data/name).symlink_to(server/'data'/name,target_is_directory=True)
 enchant=data/'game/ItemEnchant.ini'
 text=enchant.read_text(encoding='utf-8')
 # The rate table uses the current enchant level. The fixture starts at +5.
 text,count=re.subn(r'(?m)^WeaponStep05[ 	]*=.*$', 'WeaponStep05=1000000',text,count=1)
 if count!=1:raise RuntimeError('missing WeaponStep05 in the test enchant INI')
 enchant.write_text(text,encoding='utf-8')
 env['RAPTURE_DATA_DIR']=str(data)
 env['PATH']=str(Path.home()/'.local/opt/go-1.27.4/bin')+':'+env['PATH']
 for port in (40800,40810):
  with socket.socket() as check:
   if check.connect_ex(('127.0.0.1',port))==0:raise RuntimeError('test port is already in use: '+str(port))
 users=[]
 with sqlite3.connect(db) as c:
  for _ in range(2):
   name='測'+secrets.token_hex(5);password='密'+secrets.token_hex(5);uid=c.execute('SELECT MAX(UserUID)+1 FROM PS_UserData_Users_Master').fetchone()[0]
   c.execute("INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(?,?,?,0,0,0,0,0,'N',0)",(uid,name,password));users.append((name,password,uid))
 try:
  for name,port in [('gx_login',40800),('gx_world',40810)]:
   log=open(client/('work/sqlite-'+name+'.log'),'w')
   process=subprocess.Popen(['nice',str(server/'build'/name)],cwd=temp,env=env,stdout=log,stderr=subprocess.STDOUT);children.append(process)
   deadline=time.monotonic()+30
   while True:
    with socket.socket() as s:
     if s.connect_ex(('127.0.0.1',port))==0:break
    if process.poll() is not None or time.monotonic()>deadline:raise RuntimeError('server startup failed: '+name)
    time.sleep(.1)
  for name,password,uid in users:
   account_env=dict(env,SHAIYA_TEST_USER=name,SHAIYA_PASSWORD=password)
   for test,extra in [('TestFactionAgainstRapture',{'SHAIYA_TEST_FACTION':'1'}),('TestCreationAgainstRapture',{'SHAIYA_TEST_CREATE':'試🧙'+secrets.token_hex(4)})]:
    subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^'+test+'$','-count=1','-v'],cwd=client,env=dict(account_env,**extra),check=True,timeout=60)
  name,password,uid=users[0]
  with sqlite3.connect(db) as c:
   char_id,job,level,grow=c.execute('SELECT CharID,Job,Level,Grow FROM PS_GameData_Chars WHERE UserUID=? AND Del=0',(uid,)).fetchone()
   c.execute('UPDATE PS_GameData_Chars SET Job=0,Level=43,Grow=1 WHERE CharID=?',(char_id,))
   c.execute('INSERT INTO PS_GameData_CharQuests(CharID,QuestID,Delay,Count1,Count2,Count3,Success,Finish,Del) VALUES(?,844,0,0,0,0,1,1,0)',(char_id,))
   item_uid=c.execute('SELECT COALESCE(MAX(ItemUID),0)+1 FROM PS_GameData_CharItems').fetchone()[0]
   c.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,29001,?,29,1,1,20,0,0,0,0,0,0,0,'',1,datetime('now'),'N2',0)",(char_id,item_uid))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestQuestItemOfferAgainstGX$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_USER=name,SHAIYA_PASSWORD=password,SHAIYA_TEST_ITEM_QUEST='1'),check=True,timeout=60)
  with sqlite3.connect(db) as c:
   if c.execute('SELECT Finish,Success FROM PS_GameData_CharQuests WHERE CharID=? AND QuestID=845',(char_id,)).fetchone() != (1,0):
    raise RuntimeError('quest 845 abandonment did not persist')
   c.execute('DELETE FROM PS_GameData_CharQuests WHERE CharID=? AND QuestID IN (844,845)',(char_id,))
   c.execute('UPDATE PS_GameData_Chars SET Job=?,Level=?,Grow=? WHERE CharID=?',(job,level,grow,char_id))
  with sqlite3.connect(db) as c:
   for n,(_,_,uid) in enumerate(users):
    c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=?,PosY=78,PosZ=1762,Level=40,SkillPoint=100 WHERE UserUID=?',(578+n*2,uid))
   c.execute('UPDATE PS_GameData_Chars SET Family=0,Job=5 WHERE UserUID=?',(users[1][2],))
   c.execute('UPDATE PS_GameData_Chars SET HP=500 WHERE UserUID=?',(users[0][2],))
  with sqlite3.connect(db) as c:
   uid=users[0][2]
   char_id=c.execute('SELECT CharID FROM PS_GameData_Chars WHERE UserUID=? AND Del=0',(uid,)).fetchone()[0]
   c.execute('DELETE FROM PS_GameData_CharItems WHERE CharID=? AND Bag=1 AND Slot=23',(char_id,))
   item_uid=c.execute('SELECT COALESCE(MAX(ItemUID),0)+1 FROM PS_GameData_CharItems').fetchone()[0]
   c.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,25004,?,25,4,1,23,0,0,0,0,0,0,0,'',5,datetime('now'),'N2',0)",(char_id,item_uid))
   c.execute('DELETE FROM PS_GameData_CharItems WHERE CharID=? AND Bag=1 AND Slot=22',(char_id,))
   c.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,24001,?,24,1,1,22,0,0,0,0,0,0,0,'',1,datetime('now'),'N2',0)",(char_id,item_uid+1))
   # Enchant fixtures belong only to this disposable database.
   c.execute('DELETE FROM PS_GameData_CharItems WHERE CharID=? AND Bag=0 AND Slot=5',(char_id,))
   c.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,1001,?,1,1,0,5,1200,0,0,0,0,0,0,'00000000000000000005',1,datetime('now'),'N2',0)",(char_id,item_uid+2))
   c.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,1001,?,1,1,1,21,1200,0,0,0,0,0,0,'00000000000000000000',1,datetime('now'),'N2',0)",(char_id,item_uid+3))
   c.execute('UPDATE PS_GameData_Chars SET Money=1000000000 WHERE CharID=?',(char_id,))
   weapon=c.execute('SELECT Type,TypeID FROM PS_GameData_CharItems WHERE CharID=? AND Bag=0 AND Slot=5',(char_id,)).fetchone()
   material=c.execute('SELECT Reqluc FROM PS_GameDefs_Items WHERE Type=? AND TypeID=?',weapon).fetchone()[0]
   for offset,(slot,item_id,count) in enumerate(((18,material,100),(19,95002,50)),4):
    c.execute('DELETE FROM PS_GameData_CharItems WHERE CharID=? AND Bag=1 AND Slot=?',(char_id,slot))
    c.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,?,?,?,?,1,?,0,0,0,0,0,0,0,'',?,datetime('now'),'N2',0)",(char_id,item_id,item_uid+offset,item_id//1000,item_id%1000,slot,count))
  env.update(SHAIYA_TEST_USER=users[0][0],SHAIYA_PASSWORD=users[0][1],SHAIYA_TEST_PARTNER=users[1][0],SHAIYA_TEST_PARTNER_PASSWORD=users[1][1])
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestNativeControlsAgainstRapture$','-count=1','-v'],cwd=client,env=env,check=True,timeout=120)
  # Approach runs near real mobs, with actual elapsed movement and no forged teleports.
  with sqlite3.connect(db) as c:
   for n,(_,_,uid) in enumerate(users):c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=?,PosY=45,PosZ=1671.7 WHERE UserUID=?',(799.4+n*2,uid))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestTargetEffectsAgainstIsolatedGX$','-count=1','-v'],cwd=client,env=env,check=True,timeout=60)
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestNativeControlsAgainstRapture$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_APPROACH_ONLY='1'),check=True,timeout=120)
  with sqlite3.connect(db) as c:
   for n,(_,_,uid) in enumerate(users):c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=?,PosY=78,PosZ=1762 WHERE UserUID=?',(578+n*2,uid))
  # Native controls may consume or merge the earlier potion fixture.
  with sqlite3.connect(db) as c:
   char_id=c.execute('SELECT CharID FROM PS_GameData_Chars WHERE UserUID=? AND Del=0',(users[0][2],)).fetchone()[0]
   c.execute('DELETE FROM PS_GameData_CharItems WHERE CharID=? AND Bag=1 AND Slot=23',(char_id,))
   item_uid=c.execute('SELECT COALESCE(MAX(ItemUID),0)+1 FROM PS_GameData_CharItems').fetchone()[0]
   c.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,25004,?,25,4,1,23,0,0,0,0,0,0,0,'',10,datetime('now'),'N2',0)",(char_id,item_uid))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestGroundModelPickupAgainstRapture$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_PICKUP='1'),check=True,timeout=60)
  with sqlite3.connect(db) as c:
   c.execute('UPDATE PS_GameData_Chars SET Family=0,Job=5,MP=1000,SP=500,SkillPoint=100 WHERE UserUID=?',(users[0][2],))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestFrozenIceAreaAgainstRapture$','-count=1','-v'],cwd=client,env=env,check=True,timeout=60)
  with sqlite3.connect(db) as c:
   c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=799.4,PosY=45,PosZ=1671.7,MP=1000,SP=500,SkillPoint=100 WHERE UserUID=?',(users[1][2],))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestFrozenIceAreaAgainstRapture$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_USER=users[1][0],SHAIYA_PASSWORD=users[1][1],SHAIYA_TEST_FROZEN_VICTIMS='1'),check=True,timeout=60)
  with sqlite3.connect(db) as c:
   chars=[c.execute('SELECT CharID,CharName FROM PS_GameData_Chars WHERE UserUID=? AND Del=0',(u[2],)).fetchone() for u in users]
   guild_id=c.execute('SELECT COALESCE(MAX(GuildID),0)+1 FROM PS_GameData_Guilds').fetchone()[0]
   c.execute("INSERT INTO PS_GameData_Guilds(GuildID,GuildName,MasterUserID,MasterCharID,MasterName,Country,TotalCount,GuildPoint,CreateDate) VALUES(?,'N2NativeGuild',?,?,?,0,2,0,datetime('now'))",(guild_id,users[0][0],chars[0][0],chars[0][1]))
   c.execute("INSERT INTO PS_GameData_GuildDetails(GuildID,Etin,Remark) VALUES(?,1234,'Original notice')",(guild_id,))
   for n,(char_id,_) in enumerate(chars):c.execute("INSERT INTO PS_GameData_GuildChars(GuildID,CharID,GuildLevel,JoinDate) VALUES(?,?,?,datetime('now'))",(guild_id,char_id,1 if n==0 else 9))
   c.execute('INSERT INTO PS_GameData_CharQuests(CharID,QuestID,Delay,Count1,Count2,Count3,Success,Finish,Del) VALUES(?,1,0,0,0,0,0,0,0)',(chars[0][0],))
  # Refresh only the isolated world's startup guild cache, after its test
  # sessions have disconnected. Personal services and their DB are untouched.
  children[-1].terminate();children[-1].wait(timeout=10)
  log=open(client/'work/sqlite-gx_world.log','a')
  process=subprocess.Popen(['nice',str(server/'build/gx_world')],cwd=temp,env=env,stdout=log,stderr=subprocess.STDOUT);children.append(process)
  deadline=time.monotonic()+30
  while True:
   with socket.socket() as s:
    if s.connect_ex(('127.0.0.1',40810))==0:break
   if process.poll() is not None or time.monotonic()>deadline:raise RuntimeError('isolated guild world startup failed')
   time.sleep(.1)
  for test in ('TestGuildNoticeAndEtinAgainstRapture','TestNativeGuildQuestButtonsAgainstRapture'):
   subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^'+test+'$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_GUILD='1'),check=True,timeout=60)
  with sqlite3.connect(db) as c:
   c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=604,PosY=78,PosZ=1720,Money=50000 WHERE UserUID=?',(users[0][2],))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestGatekeeperAgainstRapture$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_GATEKEEPER='1'),check=True,timeout=60)
  with sqlite3.connect(db) as c:
   c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=943,PosY=13,PosZ=777,Money=50000 WHERE UserUID=?',(users[0][2],))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestGatekeeperCrossMapAgainstRapture$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_GATEKEEPER_PAID='1'),check=True,timeout=90)
  # A real duel uses only these temporary fixtures and verifies live 0x020E counters.
  with sqlite3.connect(db) as c:
   for n,(_,_,uid) in enumerate(users):
    c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=?,PosY=78,PosZ=1762,Family=0,Job=0,Level=40,Str=2000,Dex=2000,HP=?,MP=1000,SP=500,Money=50000 WHERE UserUID=?',(578+n*2,500 if n==0 else 1,uid))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestDuelVictoryWagersAgainstRapture$','-count=1','-v'],cwd=client,env=env,check=True,timeout=90)
  # Moving jumps must land without rewinding to a stale horizontal snapshot.
  with sqlite3.connect(db) as c:
   c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=578,PosY=78,PosZ=1762,HP=10000 WHERE UserUID=?',(users[0][2],))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestMovingJumpAgainstIsolatedGX$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_MOVING_JUMP='1'),check=True,timeout=60)
  # Mob navigation uses a real dungeon and authoritative XYZ packets.
  with sqlite3.connect(db) as c:
   c.execute('UPDATE PS_GameData_Chars SET Map=3,PosX=230,PosY=13.24121,PosZ=60,HP=10000,MP=1000,SP=500 WHERE UserUID=?',(users[0][2],))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestMobNavigationDungeonAgainstRapture$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_MOB_NAVIGATION='1'),check=True,timeout=60)
  # Native on-foot speed: Sonic boots, haste, candy and a late observer.
  with sqlite3.connect(db) as c:
   for n,(_,_,uid) in enumerate(users):
    c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=?,PosY=78,PosZ=1762,Family=0,Job=2,Level=40,HP=10000,MP=1000,SP=1000 WHERE UserUID=?',(578+n*2,uid))
    char_id=c.execute('SELECT CharID FROM PS_GameData_Chars WHERE UserUID=? AND Del=0',(uid,)).fetchone()[0]
    c.execute('DELETE FROM PS_GameData_CharApplySkills WHERE CharID=?',(char_id,))
    c.execute('DELETE FROM PS_GameData_CharSkills WHERE CharID=?',(char_id,))
    c.execute('DELETE FROM PS_GameData_CharItems WHERE CharID=?',(char_id,))
    if n==0:
     c.execute('INSERT INTO PS_GameData_CharSkills(CharID,SkillID,SkillLevel,Number,Delay,Del) VALUES(?,114,1,40,0,0)',(char_id,))
     item_uid=c.execute('SELECT COALESCE(MAX(ItemUID),0)+1 FROM PS_GameData_CharItems').fetchone()[0]
     for item_type,type_id,bag,slot,count,gem in ((21,138,0,4,1,106),(25,177,1,19,3,0)):
      c.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,?,?,?,?,?,?,0,?,0,0,0,0,0,'',?,datetime('now'),'N2',0)",(char_id,item_type*1000+type_id,item_uid,item_type,type_id,bag,slot,gem,count));item_uid+=1
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestPlayerSpeedAgainstIsolatedGX$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_PLAYER_SPEED='1'),check=True,timeout=90)
  # Attack cadence uses opposing factions, native Flash/mastery/ring and haste.
  with sqlite3.connect(db) as c:
   for n,(_,_,uid) in enumerate(users):
    c.execute('UPDATE PS_GameData_UserMaxGrow SET Country=? WHERE UserUID=?',(n,uid))
    c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=?,PosY=78,PosZ=1762,Family=?,Job=0,Level=60,Grow=1,Str=40,Dex=1000,Rec=2000,HP=20000,MP=1000,SP=1000 WHERE UserUID=?',(578+n*2,0 if n==0 else 2,uid))
    char_id=c.execute('SELECT CharID FROM PS_GameData_Chars WHERE UserUID=? AND Del=0',(uid,)).fetchone()[0]
    c.execute('DELETE FROM PS_GameData_CharApplySkills WHERE CharID=?',(char_id,))
    c.execute('DELETE FROM PS_GameData_CharSkills WHERE CharID=?',(char_id,))
    c.execute('DELETE FROM PS_GameData_CharItems WHERE CharID=?',(char_id,))
    if n==0:
     for skill,number in ((15,40),(129,41)):c.execute('INSERT INTO PS_GameData_CharSkills(CharID,SkillID,SkillLevel,Number,Delay,Del) VALUES(?,?,1,?,0,0)',(char_id,skill,number))
     item_uid=c.execute('SELECT COALESCE(MAX(ItemUID),0)+1 FROM PS_GameData_CharItems').fetchone()[0]
     for item_type,type_id,bag,slot,gem in ((1,1,0,5,64),(22,56,1,20,0)):
      c.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,?,?,?,?,?,?,0,?,0,0,0,0,0,'',1,datetime('now'),'N2',0)",(char_id,item_type*1000+type_id,item_uid,item_type,type_id,bag,slot,gem));item_uid+=1
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestAttackSpeedAgainstIsolatedGX$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_ATTACK_SPEED='1'),check=True,timeout=90)
  # Basics and spells share the Ext. Fast recovery; exercise the client queue too.
  with sqlite3.connect(db) as c:
   char_id=c.execute('SELECT CharID FROM PS_GameData_Chars WHERE UserUID=? AND Del=0',(users[0][2],)).fetchone()[0]
   assert not c.execute('SELECT 1 FROM PS_GameData_CharItems WHERE CharID=? AND Bag=0 AND Slot=9 AND Del=0',(char_id,)).fetchone()
   for skill,number in ((30,42),(31,43)):
    c.execute('INSERT INTO PS_GameData_CharSkills(CharID,SkillID,SkillLevel,Number,Delay,Del) VALUES(?,?,1,?,0,0)',(char_id,skill,number))
   item_uid=c.execute('SELECT COALESCE(MAX(ItemUID),0)+1 FROM PS_GameData_CharItems').fetchone()[0]
   c.execute("INSERT INTO PS_GameData_CharItems(CharID,ItemID,ItemUID,Type,TypeID,Bag,Slot,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,Count,Maketime,Maketype,Del) VALUES(?,22213,?,22,213,0,9,0,0,0,0,0,0,0,'',1,datetime('now'),'N2_SHARED',0)",(char_id,item_uid))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestSharedCombatSpeedAgainstIsolatedGX$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_SHARED_SPEED='1'),check=True,timeout=60)
  with sqlite3.connect(db) as c:
   c.execute("DELETE FROM PS_GameData_CharItems WHERE CharID=? AND Maketype='N2_SHARED'",(char_id,))
   c.execute('DELETE FROM PS_GameData_CharSkills WHERE CharID=? AND SkillID IN(30,31) AND Number IN(42,43)',(char_id,))
   c.execute('DELETE FROM PS_GameData_CharApplySkills WHERE CharID=? AND SkillID IN(30,31)',(char_id,))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestAttackMovementAgainstIsolatedGX$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_ATTACK_MOVEMENT='1'),check=True,timeout=60)
  # Map clicks carry normalized X/Z only; the server resolves outdoor/dungeon Y.
  with sqlite3.connect(db) as c:
   c.execute('UPDATE PS_UserData_Users_Master SET Admin=1 WHERE UserUID=?',(users[0][2],))
   c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=578,PosY=78,PosZ=1762,HP=10000 WHERE UserUID=?',(users[0][2],))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestGMMapTeleportAgainstIsolatedGX$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_GM_MAP='1'),check=True,timeout=90)
  # Functional native ladder: the first map-1 ivy guide, approached from its front.
  with sqlite3.connect(db) as c:
   c.execute('UPDATE PS_GameData_Chars SET Map=1,PosX=1685.1454,PosY=5.1,PosZ=619.3524,HP=20000 WHERE UserUID=?',(users[0][2],))
  subprocess.run(['nice','go','test','-tags','integration','./internal/app','-run','^TestLaddersAgainstIsolatedGX$','-count=1','-v'],cwd=client,env=dict(env,SHAIYA_TEST_LADDER='1'),check=True,timeout=90)
 finally:
  for process in reversed(children):
   process.terminate()
   try:process.wait(timeout=10)
   except subprocess.TimeoutExpired:process.kill();process.wait()
 print('N2 SQLite: isolated servers stopped, temporary database and accounts removed')
