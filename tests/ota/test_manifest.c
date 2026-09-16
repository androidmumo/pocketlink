#include <assert.h>
#include <stdio.h>
#include <string.h>
#include "pocketlink_ota.h"
int main(int argc,char **argv){
 assert(argc==2);FILE *f=fopen(argv[1],"rb");assert(f);char raw[4096];size_t n=fread(raw,1,sizeof(raw),f);fclose(f);
 pl_ota_offer offer;assert(pl_ota_parse(raw,n,0,&offer)==ESP_OK);assert(offer.sequence==7&&offer.size==288&&!strcmp(offer.version,"test-1"));
 assert(pl_ota_parse(raw,n,7,&offer)==ESP_ERR_INVALID_VERSION);assert(pl_ota_parse(raw,n,8,&offer)==ESP_ERR_INVALID_VERSION);
 assert(pl_ota_parse(raw,n-5,0,&offer)!=ESP_OK);raw[n]='x';assert(pl_ota_parse(raw,n+1,0,&offer)!=ESP_OK);
 raw[20]=raw[20]=='A'?'B':'A';assert(pl_ota_parse(raw,n,0,&offer)!=ESP_OK);
 assert(pl_ota_parse(NULL,0,0,&offer)!=ESP_OK);
 puts("Device manifest parser: OpenSSL signature interoperability, replay and tamper rejection PASS");
}
