import argparse, datetime, glob, json, math, os, re, subprocess, sys, tempfile
import pandas as pd
import pyreadstat

here=os.path.dirname(os.path.abspath(__file__))
parser=argparse.ArgumentParser(description='Compare SAS files against pyreadstat')
parser.add_argument('directory', help='directory containing SAS7BDAT files (searched recursively)')
parser.add_argument('--reader', help='prebuilt Go dump executable (default: build current checkout)')
args=parser.parse_args()
files=sorted(glob.glob(os.path.join(args.directory,'**','*.sas7bdat'),recursive=True))
if not files:
 sys.exit('no .sas7bdat files found')
tmp=tempfile.TemporaryDirectory()
reader=args.reader or os.path.join(tmp.name,'reader')
if not args.reader:
 subprocess.run(['go','build','-o',reader,'./parity'],cwd=os.path.dirname(here),check=True)

def check(path,exe):
 try:
  df,meta=pyreadstat.read_sas7bdat(path, disable_datetime_conversion=True)
 except Exception as e:
  if 'invalid byte sequence' in str(e):
   p=subprocess.run([exe,path],stdout=subprocess.DEVNULL,stderr=subprocess.PIPE,text=True)
   if p.returncode and 'invalid Windows-1252 byte' in p.stderr: return []
  return ['pyreadstat: '+str(e)]
 p=subprocess.Popen([exe,path],stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
 issues=[]
 try:
  line=p.stdout.readline()
  if not line:
   err=p.stderr.read(); p.wait()
   return ['reader failed: '+err[-400:].replace('\n',' ')]
  m=json.loads(line)
  if m['rows']!=len(df): issues.append('row count')
  if m['names']!=meta.column_names: issues.append('names')
  labels=[x or '' for x in meta.column_labels]
  if (m['labels'] or [])!=labels: issues.append('labels')
  # The Go API exposes the format name but not its width or trailing dot.
  fmts=[re.sub(r'\d*(?:\.\d*)?$', '', meta.original_variable_types.get(n) or '') for n in meta.column_names]
  if (m['formats'] or [])!=fmts: issues.append('formats')
  types=[1 if meta.readstat_variable_types[n]=='string' else 0 for n in meta.column_names]
  if (m['types'] or [])!=types: issues.append('types')
  count=0; mismatch=None
  for row in df.itertuples(index=False,name=None):
   line=p.stdout.readline()
   if not line: issues.append('short output'); break
   got=json.loads(line); count+=1
   if len(got)!=len(row): mismatch=mismatch or f'row width {count}'; continue
   for j,(a,b) in enumerate(zip(got,row)):
    if pd.isna(b): equal=a is None or (isinstance(a,str) and not a)
    elif isinstance(b,str): equal=isinstance(a,str) and a.rstrip(' ')==b.rstrip(' ')
    elif isinstance(a,str) and fmts[j] in ('DATE','MMDDYY','DATETIME'):
     value=datetime.datetime.fromisoformat(a.replace('Z','+00:00')).replace(tzinfo=None)
     seconds=(value-datetime.datetime(1960,1,1)).total_seconds()
     value=seconds if fmts[j]=='DATETIME' else seconds/86400
     equal=math.isclose(value,float(b),rel_tol=0,abs_tol=1e-9)
    else: equal=isinstance(a,(int,float)) and math.isclose(a,float(b),rel_tol=0,abs_tol=1e-9)
    if not equal and mismatch is None: mismatch=f'cell row {count} col {meta.column_names[j]}: {a!r} != {b!r}'
  if mismatch: issues.append(mismatch)
  if count != len(df): issues.append('rows emitted')
  p.stdout.close(); err=p.stderr.read(); rc=p.wait()
  if rc: issues.append('reader failed: '+err[-400:].replace('\n',' '))
 except Exception as e:
  p.kill(); p.wait(); issues.append('reader failed: '+str(e))
 return issues

DATE_FORMATS=set('WEEKDATE MMDDYY DDMMYY YYMMDD DATE DDMMYYB DDMMYYC DDMMYYD DDMMYYN DDMMYYP DDMMYYS MMDDYYB MMDDYYC MMDDYYD MMDDYYN MMDDYYP MMDDYYS WEEKDATX DTDATE IS8601DA E8601DA B8601DA YYMMDDB YYMMDDD YYMMDDN YYMMDDP YYMMDDS'.split())
DATETIME_FORMATS=set('DATETIME E8601DT DATEAMPM MDYAMPM IS8601DT B8601DT B8601DN'.split())
TIME_FORMATS=set('TIME HHMM TOD TIMEAMPM IS8601TM E8601TM B8601TM'.split())

def civil_date(days):
 # Proleptic Gregorian date for integer days since 1960; unlike datetime,
 # this supports the huge DTDATE day count that makes pyreadstat overflow.
 z=days-3653+719468
 era=z//146097
 doe=z-era*146097
 yoe=(doe-doe//1460+doe//36524-doe//146096)//365
 y=yoe+era*400
 doy=doe-(365*yoe+yoe//4-yoe//100)
 mp=(5*doy+2)//153
 day=doy-(153*mp+2)//5+1
 month=mp+3 if mp<10 else mp-9
 y+=month<=2
 return f'{y:04d}-{month:02d}-{day:02d}'

def check_overflow_dates(path,exe):
 df,meta=pyreadstat.read_sas7bdat(path,disable_datetime_conversion=True)
 p=subprocess.Popen([exe,path],stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True,env={**os.environ,'SAS_DATES':'1'})
 try:
  m=json.loads(p.stdout.readline()); issues=[]
  for i,row in enumerate(df.itertuples(index=False,name=None)):
   got=json.loads(p.stdout.readline())
   for j,(a,b) in enumerate(zip(got,row)):
    fmt=m['formats'][j]
    if pd.isna(b): equal=a is None
    elif fmt in DATE_FORMATS: equal=isinstance(a,str) and a.split('T')[0]==civil_date(int(b))
    elif fmt in DATETIME_FORMATS:
     whole=math.floor(float(b))
     expected=datetime.datetime(1960,1,1)+datetime.timedelta(seconds=whole,microseconds=round((float(b)-whole)*1000000))
     equal=isinstance(a,str) and datetime.datetime.fromisoformat(a.replace('Z','+00:00')).replace(tzinfo=None)==expected
    elif isinstance(b,str): equal=isinstance(a,str) and a.rstrip(' ')==b.rstrip(' ')
    else: equal=isinstance(a,(int,float)) and math.isclose(a,float(b),rel_tol=0,abs_tol=1e-9)
    if not equal and not issues: issues.append(f'row {i+1} {m["names"][j]}: {a!r} != raw {b!r} ({fmt})')
  p.stdout.close(); err=p.stderr.read(); rc=p.wait()
  if rc: issues.append('reader failed '+err[-300:])
  return issues
 except Exception as e:
  p.kill(); p.wait(); return [repr(e)]

def check_dates(path,exe):
 try:
  df,meta=pyreadstat.read_sas7bdat(path)
 except Exception as e:
  if 'invalid byte sequence' in str(e): return check(path,exe)
  if isinstance(e,OverflowError): return check_overflow_dates(path,exe)
  return ['pyreadstat conversion: '+repr(e)]
 p=subprocess.Popen([exe,path],stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True,env={**os.environ,'SAS_DATES':'1'})
 try:
  m=json.loads(p.stdout.readline()); issues=[]
  for i,row in enumerate(df.itertuples(index=False,name=None)):
   got=json.loads(p.stdout.readline())
   for j,(a,b) in enumerate(zip(got,row)):
    if pd.isna(b): equal=a is None or (isinstance(a,str) and not a)
    elif isinstance(b,datetime.time):
     seconds=(math.floor(float(a))*1000000+round((float(a)-math.floor(float(a)))*1000000)) % 86400000000 if isinstance(a,(int,float)) else None
     expected=((b.hour*60+b.minute)*60+b.second)*1000000+b.microsecond
     equal=seconds==expected
    elif isinstance(b,datetime.datetime):
     actual=datetime.datetime.fromisoformat(a.replace('Z','+00:00')).replace(tzinfo=None) if isinstance(a,str) else None
     equal=actual==b.replace(tzinfo=None)
    elif isinstance(b,datetime.date):
     actual=datetime.datetime.fromisoformat(a.replace('Z','+00:00')).date() if isinstance(a,str) else None
     equal=actual==b
    elif isinstance(b,str): equal=isinstance(a,str) and a.rstrip(' ')==b.rstrip(' ')
    else: equal=isinstance(a,(int,float)) and math.isclose(a,float(b),rel_tol=0,abs_tol=1e-9)
    if not equal and not issues: issues.append(f'row {i+1} {m["names"][j]}: {a!r} != {b!r}')
  p.stdout.close(); err=p.stderr.read(); rc=p.wait()
  if rc: issues.append('reader failed '+err[-300:])
  return issues
 except Exception as e:
  p.kill();p.wait();return [repr(e)]

rawpass=convertedpass=0
for path in files:
 raw=check(path,reader)
 converted=check_dates(path,reader)
 rawpass+=not raw
 convertedpass+=not converted
 print(path, 'raw='+('PASS' if not raw else repr(raw)), 'converted='+('PASS' if not converted else repr(converted)),flush=True)
print(f'TOTAL raw={rawpass}/{len(files)} converted={convertedpass}/{len(files)}',flush=True)
sys.exit(0 if rawpass == len(files) and convertedpass == len(files) else 1)
