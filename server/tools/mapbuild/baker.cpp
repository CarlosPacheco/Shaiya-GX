// GX offline navigation baker. Uses Recast/Detour, copyright Mikko Mononen,
// distributed under the zlib license (see THIRD_PARTY_NOTICES.md).
#include <Recast.h>
#include <DetourNavMesh.h>
#include <DetourNavMeshBuilder.h>
#include <ChunkyTriMesh.h>
#include <algorithm>
#include <cmath>
#include <cstdlib>
#include <cstdint>
#include <fstream>
#include <iostream>
#include <memory>
#include <sstream>
#include <stdexcept>
#include <vector>

static void check(bool ok,const char* message){if(!ok)throw std::runtime_error(message);}
static int power2(int n){int x=1;while(x<n)x*=2;return x;}
static void write32(std::ofstream& out,uint32_t n){out.write(reinterpret_cast<char*>(&n),4);}
int main(int argc,char** argv){try{
 check(argc==4||argc==5,"usage: gx_recast_baker input.obj output.gxnav cell-size");
 std::ifstream input(argv[1]);check(bool(input),"cannot open geometry");
 std::vector<float> verts;std::vector<int> tris;std::string line;
 while(std::getline(input,line)){std::istringstream row(line);std::string tag;row>>tag;if(tag=="v"){float a,b,c;check(bool(row>>a>>b>>c),"invalid vertex");verts.insert(verts.end(),{a,b,c});}else if(tag=="f"){int a,b,c;check(bool(row>>a>>b>>c),"invalid face");tris.insert(tris.end(),{a-1,b-1,c-1});}}
 check(!tris.empty(),"empty geometry");
 uint32_t maskSize=0;std::vector<unsigned char> mask;
 if(argc==5){std::ifstream in(argv[4],std::ios::binary);in.read(reinterpret_cast<char*>(&maskSize),4);check(maskSize>0&&maskSize<=16384,"invalid walk mask size");mask.resize(uint64_t(maskSize)*maskSize/8);in.read(reinterpret_cast<char*>(mask.data()),mask.size());check(bool(in),"truncated walk mask");}
 rcContext ctx;rcChunkyTriMesh chunky;check(rcCreateChunkyTriMesh(verts.data(),tris.data(),tris.size()/3,256,&chunky),"cannot partition geometry");
 float bmin[3],bmax[3];rcCalcBounds(verts.data(),verts.size()/3,bmin,bmax);
 const float cs=std::stof(argv[3]),ch=.1f;const char* requestedTile=std::getenv("GX_RECAST_TILE_SIZE");const int tileSize=requestedTile?std::stoi(requestedTile):128;check(tileSize==128||tileSize==256||tileSize==512,"invalid tile size");
 int gw,gh;rcCalcGridSize(bmin,bmax,cs,&gw,&gh);int tw=(gw+tileSize-1)/tileSize,th=(gh+tileSize-1)/tileSize;
 int maxTiles=power2(tw*th);check(maxTiles<=16384,"map exceeds tile capacity");
 dtNavMeshParams np{};std::copy(bmin,bmin+3,np.orig);np.tileWidth=np.tileHeight=tileSize*cs;np.maxTiles=maxTiles;np.maxPolys=(1<<22)/maxTiles;
 std::unique_ptr<dtNavMesh,decltype(&dtFreeNavMesh)> nav(dtAllocNavMesh(),dtFreeNavMesh);check(nav&&dtStatusSucceed(nav->init(&np)),"cannot initialize navigation");
 int polygonCount=0;
 for(int ty=0;ty<th;++ty)for(int tx=0;tx<tw;++tx){
  rcConfig c{};c.cs=cs;c.ch=ch;c.walkableSlopeAngle=50;c.walkableHeight=(int)ceilf(1.8f/ch);c.walkableClimb=(int)floorf(.5f/ch);c.walkableRadius=(int)ceilf(.3f/cs);c.maxEdgeLen=(int)(12/cs);c.maxSimplificationError=.5f;c.minRegionArea=4;c.mergeRegionArea=64;c.maxVertsPerPoly=6;c.tileSize=tileSize;c.borderSize=c.walkableRadius+3;c.width=c.height=tileSize+2*c.borderSize;c.detailSampleDist=cs*6;c.detailSampleMaxError=ch*.5f;
  std::copy(bmin,bmin+3,c.bmin);std::copy(bmax,bmax+3,c.bmax);c.bmin[0]=bmin[0]+tx*tileSize*cs-c.borderSize*cs;c.bmin[2]=bmin[2]+ty*tileSize*cs-c.borderSize*cs;c.bmax[0]=bmin[0]+(tx+1)*tileSize*cs+c.borderSize*cs;c.bmax[2]=bmin[2]+(ty+1)*tileSize*cs+c.borderSize*cs;
  float qmin[]={c.bmin[0],c.bmin[2]},qmax[]={c.bmax[0],c.bmax[2]};std::vector<int> chunks(chunky.nnodes);int count=rcGetChunksOverlappingRect(&chunky,qmin,qmax,chunks.data(),chunks.size());if(!count)continue;
  std::unique_ptr<rcHeightfield,decltype(&rcFreeHeightField)> solid(rcAllocHeightfield(),rcFreeHeightField);check(solid&&rcCreateHeightfield(&ctx,*solid,c.width,c.height,c.bmin,c.bmax,cs,ch),"heightfield failed");
  std::vector<unsigned char> areas(chunky.maxTrisPerChunk);
  for(int j=0;j<count;++j){const auto& node=chunky.nodes[chunks[j]];const int* ct=&chunky.tris[node.i*3];std::fill(areas.begin(),areas.end(),0);rcMarkWalkableTriangles(&ctx,c.walkableSlopeAngle,verts.data(),verts.size()/3,ct,node.n,areas.data());check(rcRasterizeTriangles(&ctx,verts.data(),verts.size()/3,ct,areas.data(),node.n,*solid,c.walkableClimb),"rasterization failed");}
  rcFilterLowHangingWalkableObstacles(&ctx,c.walkableClimb,*solid);rcFilterLedgeSpans(&ctx,c.walkableHeight,c.walkableClimb,*solid);rcFilterWalkableLowHeightSpans(&ctx,c.walkableHeight,*solid);
  std::unique_ptr<rcCompactHeightfield,decltype(&rcFreeCompactHeightfield)> compact(rcAllocCompactHeightfield(),rcFreeCompactHeightfield);check(compact&&rcBuildCompactHeightfield(&ctx,c.walkableHeight,c.walkableClimb,*solid,*compact),"compact field failed");solid.reset();
  if(maskSize){for(int z=0;z<compact->height;++z)for(int x=0;x<compact->width;++x){int wx=(int)floorf(compact->bmin[0]+(x+.5f)*cs),wz=(int)floorf(compact->bmin[2]+(z+.5f)*cs);bool blocked=wx<0||wz<0||wx>=(int)maskSize||wz>=(int)maskSize;if(!blocked){uint64_t bit=uint64_t(wz)*maskSize+wx;blocked=(mask[bit/8]&(1u<<(bit%8)))!=0;}if(blocked){const auto& cell=compact->cells[x+z*compact->width];for(unsigned int k=cell.index;k<cell.index+cell.count;++k)compact->areas[k]=RC_NULL_AREA;}}}
  check(rcErodeWalkableArea(&ctx,c.walkableRadius,*compact),"erosion failed");check(rcBuildDistanceField(&ctx,*compact),"distance field failed");check(rcBuildRegions(&ctx,*compact,c.borderSize,c.minRegionArea,c.mergeRegionArea),"regions failed");
  std::unique_ptr<rcContourSet,decltype(&rcFreeContourSet)> contours(rcAllocContourSet(),rcFreeContourSet);check(contours&&rcBuildContours(&ctx,*compact,c.maxSimplificationError,c.maxEdgeLen,*contours),"contours failed");if(!contours->nconts)continue;
  std::unique_ptr<rcPolyMesh,decltype(&rcFreePolyMesh)> poly(rcAllocPolyMesh(),rcFreePolyMesh);check(poly&&rcBuildPolyMesh(&ctx,*contours,c.maxVertsPerPoly,*poly),"polygons failed");if(!poly->npolys)continue;
  std::unique_ptr<rcPolyMeshDetail,decltype(&rcFreePolyMeshDetail)> detail(rcAllocPolyMeshDetail(),rcFreePolyMeshDetail);check(detail&&rcBuildPolyMeshDetail(&ctx,*poly,*compact,c.detailSampleDist,c.detailSampleMaxError,*detail),"detail failed");
  for(int j=0;j<poly->npolys;++j){poly->areas[j]=0;poly->flags[j]=1;}
  dtNavMeshCreateParams p{};p.verts=poly->verts;p.vertCount=poly->nverts;p.polys=poly->polys;p.polyAreas=poly->areas;p.polyFlags=poly->flags;p.polyCount=poly->npolys;p.nvp=poly->nvp;p.detailMeshes=detail->meshes;p.detailVerts=detail->verts;p.detailVertsCount=detail->nverts;p.detailTris=detail->tris;p.detailTriCount=detail->ntris;p.walkableHeight=1.8f;p.walkableRadius=.3f;p.walkableClimb=.5f;p.tileX=tx;p.tileY=ty;std::copy(poly->bmin,poly->bmin+3,p.bmin);std::copy(poly->bmax,poly->bmax+3,p.bmax);p.cs=cs;p.ch=ch;p.buildBvTree=true;
  check(poly->npolys<=np.maxPolys,"tile polygon capacity exceeded");unsigned char* data=nullptr;int size=0;check(dtCreateNavMeshData(&p,&data,&size),"tile encoding failed");auto status=nav->addTile(data,size,DT_TILE_FREE_DATA,0,nullptr);if(dtStatusFailed(status)){dtFree(data);throw std::runtime_error("tile insertion failed at "+std::to_string(tx)+","+std::to_string(ty)+" polygons="+std::to_string(poly->npolys)+" limit="+std::to_string(np.maxPolys)+" status="+std::to_string(status));}polygonCount+=poly->npolys;
 }

 std::ofstream out(argv[2],std::ios::binary);check(bool(out),"cannot create navigation file");int tiles=0;for(int i=0;i<nav->getMaxTiles();++i){auto tile=static_cast<const dtNavMesh*>(nav.get())->getTile(i);if(tile->header&&tile->dataSize)++tiles;}
 write32(out,('M'<<24)|('S'<<16)|('E'<<8)|'T');write32(out,1);write32(out,tiles);out.write(reinterpret_cast<const char*>(&np),sizeof(np));
 for(int i=0;i<nav->getMaxTiles();++i){auto tile=static_cast<const dtNavMesh*>(nav.get())->getTile(i);if(!tile->header||!tile->dataSize)continue;write32(out,nav->getTileRef(tile));write32(out,tile->dataSize);out.write(reinterpret_cast<const char*>(tile->data),tile->dataSize);}
 out.close();check(bool(out),"navigation write failed");std::cout<<polygonCount<<'\n';return 0;
 }catch(const std::exception& e){std::cerr<<e.what()<<'\n';return 1;}}
