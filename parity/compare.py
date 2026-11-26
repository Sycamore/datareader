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

forkpass=0
for path in files:
 f=check(path,reader)
 forkpass+=not f
 print(path, 'PASS' if not f else repr(f),flush=True)
print(f'TOTAL {forkpass}/{len(files)}',flush=True)

def check_dates(path,exe):
 try:
  df,meta=pyreadstat.read_sas7bdat(path)
 except Exception as e: return ['pyreadstat conversion: '+repr(e)]
 p=subprocess.Popen([exe,path],stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True,env={**os.environ,'SAS_DATES':'1'})
 try:
  m=json.loads(p.stdout.readline()); issues=[]; tested=0
  for i,row in enumerate(df.itertuples(index=False,name=None)):
   got=json.loads(p.stdout.readline())
   for j,(a,b) in enumerate(zip(got,row)):
    if m['formats'][j] not in ('DATE','MMDDYY','DATETIME') or pd.isna(b): continue
    tested+=1
    actual=datetime.datetime.fromisoformat(a.replace('Z','+00:00')).replace(tzinfo=None)
    if isinstance(b,datetime.date) and not isinstance(b,datetime.datetime): b=datetime.datetime.combine(b,datetime.time())
    if abs((actual-b).total_seconds())>0.000001: issues.append(f'row {i+1} {m["names"][j]}: {actual} != {b}')
  p.stdout.close(); err=p.stderr.read(); rc=p.wait()
  if rc: issues.append('reader failed '+err[-300:])
  if not tested: issues.append('no converted dates checked')
  return issues
 except Exception as e:
  p.kill();p.wait();return [repr(e)]

for path in files:
 if os.path.basename(path) in ('date_formats.sas7bdat','datetime.sas7bdat'):
  print('CONVERTED',path,check_dates(path,reader),flush=True)
sys.exit(0 if forkpass == len(files) else 1)
