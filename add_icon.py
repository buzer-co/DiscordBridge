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

root=dirnode([3,14]);icon_dir=dirnode(list(range(1,count+1)));grp_dir=dirnode([1]);
wr(root+16,'<II',3,icon_dir|0x80000000)
wr(root+24,'<II',14,grp_dir|0x80000000)
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
