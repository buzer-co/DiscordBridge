import struct,sys,os
from pathlib import Path

source=Path(sys.argv[1]); target=Path(sys.argv[2]); ico_path=Path(sys.argv[3]); ico=ico_path.read_bytes()
reserved,icon_type,count=struct.unpack_from('<HHH',ico,0)
assert reserved==0 and icon_type==1 and 0<count<=16
icons=[]
for i in range(count):
    pos=6+i*16
    w,h,cols,rs,planes,bits,size,offset=struct.unpack_from('<BBBBHHII',ico,pos)
    icons.append((w,h,cols,rs,planes,bits,ico[offset:offset+size]))

def align(x,n):return (x+n-1)//n*n
exe=bytearray(source.read_bytes())
pe=struct.unpack_from('<I',exe,0x3c)[0]; assert exe[pe:pe+4]==b'PE\0\0'
opt=pe+24;sizeopt=struct.unpack_from('<H',exe,pe+20)[0];sectstart=opt+sizeopt
nsec=struct.unpack_from('<H',exe,pe+6)[0]; assert struct.unpack_from('<H',exe,opt)[0]==0x20b
file_align=struct.unpack_from('<I',exe,opt+36)[0];sec_align=struct.unpack_from('<I',exe,opt+32)[0]
virt=align(struct.unpack_from('<I',exe,opt+56)[0],sec_align)
assert sectstart+40*(nsec+1)<struct.unpack_from('<I',exe,opt+60)[0]
# IMAGE_RESOURCE_DIRECTORY tree
buf=bytearray()
def alloc(size,alignment=4):
    k=align(len(buf),alignment)
    if k>len(buf):buf.extend(b'\0'*(k-len(buf)))
    pos=len(buf);buf.extend(b'\0'*size);return pos

def wr(pos,fmt,*vals):struct.pack_into(fmt,buf,pos,*vals)
def dirnode(entries):
    ofs=alloc(16+8*len(entries))
    wr(ofs,'<IIHHHH',0,0,0,0,0,len(entries))
    return ofs

root=dirnode([3,14,16]);icon_dir=dirnode(list(range(1,count+1)));grp_dir=dirnode([1]);version_dir=dirnode([1]);
wr(root+16,'<II',3,icon_dir|0x80000000)
wr(root+24,'<II',14,grp_dir|0x80000000)
wr(root+32,'<II',16,version_dir|0x80000000)
icon_leaf=[]
for j in range(count):
    lang=dirnode([1033]);icon_leaf.append(lang)
    wr(icon_dir+16+j*8,'<II',j+1,lang|0x80000000)
grp_lang=dirnode([1033]);wr(grp_dir+16,'<II',1,grp_lang|0x80000000)
icon_data_entries=[]
for j in range(count):
    de=alloc(16);icon_data_entries.append(de);wr(icon_leaf[j]+16,'<II',1033,de)
grp_data_entry=alloc(16);wr(grp_lang+16,'<II',1033,grp_data_entry)
for j,icon in enumerate(icons):
    blob=icon[-1];off=alloc(len(blob));buf[off:off+len(blob)]=blob
    wr(icon_data_entries[j],'<IIII',virt+off,len(blob),0,0)
# Group icon metadata
meta=bytearray(struct.pack('<HHH',0,1,count))
for j,(w,h,colors,rs,planes,bits,raw) in enumerate(icons):
    meta+=struct.pack('<BBBBHHIH',w,h,colors,rs,planes,bits,len(raw),j+1)
off=alloc(len(meta));buf[off:off+len(meta)]=meta
wr(grp_data_entry,'<IIII',virt+off,len(meta),0,0)
# Standard VERSIONINFO resource. Added before payload construction/signing.
version=sys.argv[4] if len(sys.argv)>4 else "4.6.0.0"
parts=[int(v) for v in version.split('.')]
assert 1<=len(parts)<=4 and all(0<=v<=65535 for v in parts)
parts += [0]*(4-len(parts));version='.'.join(map(str,parts))
def vblock(key,value=b'',children=(),text=False):
    key=(key+'\0').encode('utf-16le')
    data=bytearray(struct.pack('<HHH',0,len(value)//2 if text else len(value),int(text))+key)
    data.extend(b'\0'*(align(len(data),4)-len(data)));data.extend(value)
    data.extend(b'\0'*(align(len(data),4)-len(data)))
    for child in children:
        data.extend(child);data.extend(b'\0'*(align(len(data),4)-len(data)))
    struct.pack_into('<H',data,0,len(data));return bytes(data)
strings={
    'CompanyName':'Burhan Uzer','FileDescription':'Discord Bridge connection manager',
    'FileVersion':version,'InternalName':'DiscordBridge','OriginalFilename':'DiscordBridge.exe',
    'ProductName':'Discord Bridge','ProductVersion':version,
    'LegalCopyright':'Copyright (c) 2026 Burhan Uzer',
}
entries=[vblock(k,(v+'\0').encode('utf-16le'),text=True) for k,v in strings.items()]
string_info=vblock('StringFileInfo',children=[vblock('040904b0',children=entries,text=True)],text=True)
var_info=vblock('VarFileInfo',children=[vblock('Translation',struct.pack('<HH',0x0409,1200))],text=True)
ms,ls=parts[0]<<16|parts[1],parts[2]<<16|parts[3]
fixed=struct.pack('<13I',0xFEEF04BD,0x10000,ms,ls,ms,ls,0x3f,0,0x40004,1,0,0,0)
version_blob=vblock('VS_VERSION_INFO',fixed,[string_info,var_info])
version_lang=dirnode([1033]);wr(version_dir+16,'<II',1,version_lang|0x80000000)
version_entry=alloc(16);wr(version_lang+16,'<II',1033,version_entry)
off=alloc(len(version_blob));buf[off:off+len(version_blob)]=version_blob
wr(version_entry,'<IIII',virt+off,len(version_blob),0,0)
rawptr=align(len(exe),file_align)
if rawptr>len(exe):exe.extend(b'\0'*(rawptr-len(exe)))
raw_size=align(len(buf),file_align)
exe.extend(buf);exe.extend(b'\0'*(raw_size-len(buf)))
# new section table entry
struct.pack_into('<8sIIIIIIHHI',exe,sectstart+nsec*40,b'.rsrc\x00\x00\x00',len(buf),virt,raw_size,rawptr,0,0,0,0,0x40000040)
struct.pack_into('<H',exe,pe+6,nsec+1)
struct.pack_into('<I',exe,opt+56,virt+align(len(buf),sec_align))
# Resource data dir RVA, size
struct.pack_into('<II',exe,opt+112+2*8,virt,len(buf))
# checksum not mandatory for unsigned user-mode executables
struct.pack_into('<I',exe,opt+64,0)
target.write_bytes(exe)
print('icon resources',count,'rsrc',hex(virt),'size',len(buf),'EXE size',len(exe))
