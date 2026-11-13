# RE do cliente — strings de WYD.exe

> Gerado por `node apps/tools/bin/wyd-tools.mjs strings client --bin WYD.exe` — não editar à mão.
> Offset/VA/seção são medidos pelo parser PE; a classificação por tag é heurística de regex declarada em `tools/client/strings.mjs`.

## Totais

- 4426 strings extraídas (ascii 4271, utf16le 155).
- 495 classificadas como relevantes (rede, dados, proteção, protocolo, versão, UI).

| Offset | VA | Seção | Enc | Tags | Texto |
| --- | --- | --- | --- | --- | --- |
| 0x2235a8 | 0x6235a8 | .data | ascii | protocolo | _MSG_CombineAmunlaStone |
| 0x2235d8 | 0x6235d8 | .data | ascii | protocolo | _MSG_CombineCube |
| 0x223614 | 0x623614 | .data | ascii | protocolo | _MSG_CombineGod2Item |
| 0x22362c | 0x62362c | .data | ascii | protocolo | _MSG_CombineGodUnSeal |
| 0x223544 | 0x623544 | .data | ascii | protocolo | _MSG_CombineItem |
| 0x2235c0 | 0x6235c0 | .data | ascii | protocolo | _MSG_CombineRuneStone |
| 0x2235ec | 0x6235ec | .data | ascii | protocolo | _MSG_CombineStone |
| 0x223594 | 0x623594 | .data | ascii | protocolo | _MSG_FreeMaxLevel |
| 0x223558 | 0x623558 | .data | ascii | protocolo | _MSG_MergeItem |
| 0x223534 | 0x623534 | .data | ascii | protocolo | _MSG_ReturnItem |
| 0x223568 | 0x623568 | .data | ascii | protocolo | _MSG_TransUpgrade |
| 0x22357c | 0x62357c | .data | ascii | protocolo | _MSG_TransWeaponUpgrade |
| 0x222ac8 | 0x622ac8 | .data | ascii | dados,arquivo-de-dados | ../../LimitTime.txt |
| 0x2229b0 | 0x6229b0 | .data | ascii | dados,arquivo-de-dados | ../../TMSRV/Run/AttributeMap.dat |
| 0x222990 | 0x622990 | .data | ascii | dados,arquivo-de-dados | ./Env/AttributeMap.dat |
| 0x2228c0 | 0x6228c0 | .data | ascii | dados,arquivo-de-dados | ./Guilds.txt |
| 0x21598c | 0x61598c | .data | ascii | dados,arquivo-de-dados | ./itemicon.bin |
| 0x222938 | 0x622938 | .data | ascii | dados,rede,arquivo-de-dados | ./serverlist.bin |
| 0x222398 | 0x622398 | .data | ascii | dados,arquivo-de-dados | .\ExtraItem.bin |
| 0x222290 | 0x622290 | .data | ascii | dados,arquivo-de-dados | .\ItemList.bin |
| 0x223658 | 0x623658 | .data | ascii | dados,arquivo-de-dados | .\Mixlist.bin |
| 0x2311c4 | 0x6311c4 | .data | ascii | protecao | .\XTrap\%s |
| 0x231124 | 0x631124 | .data | utf16le | protecao | .\XTrap\%S |
| 0x215d14 | 0x615d14 | .data | ascii | dados | .exp.log |
| 0x215d30 | 0x615d30 | .data | ascii | dados | .mem.log |
| 0x215aa8 | 0x615aa8 | .data | ascii | dados | \exceptions.log |
| 0x215d54 | 0x615d54 | .data | ascii | dados | \mem-leaks.log |
| 0x2222f4 | 0x6222f4 | .data | ascii | rede | #define ITEM_CHECKSUM %d |
| 0x222310 | 0x622310 | .data | ascii | rede | #define ITEM_CHECKSUM2 %d |
| 0x222270 | 0x622270 | .data | ascii | rede | #define SKILL_CHECKSUM %d |
| 0x215178 | 0x615178 | .data | ascii | dados | %s.bin |
| 0x215950 | 0x615950 | .data | ascii | dados | %s.bon |
| 0x216d3c | 0x616d3c | .data | ascii | dados | %s%02d%02d.wyt |
| 0x215958 | 0x615958 | .data | ascii | dados | %s%04d.ani |
| 0x23cb14 | 0x2787b14 | .rsrc | ascii | versao | <?xml version="1.0" encoding="UTF-8" standalone="yes"?> |
| 0x23cb4d | 0x2787b4d | .rsrc | ascii | versao | <assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0"> |
| 0x213cec | 0x613cec | .data | ascii | ui | >> Init Field Scene::End |
| 0x213bd8 | 0x613bd8 | .data | ascii | ui | >> Init Field Scene::Start |
| 0x213bbc | 0x613bbc | .data | ascii | ui | >> New Field Scene |
| 0x1fb428 | 0x5fb428 | .rdata | ascii | versao | Adobe APP14 marker: version %d, flags 0x%04x 0x%04x, transform %d |
| 0x2159d0 | 0x6159d0 | .data | ascii | dados,arquivo-de-dados | AniSound4.txt |
| 0x2229d4 | 0x6229d4 | .data | ascii | dados,arquivo-de-dados | Attributemap.dat |
| 0x222a5c | 0x622a5c | .data | ascii | dados,arquivo-de-dados | BlockIP.txt |
| 0x214c8c | 0x614c8c | .data | ascii | ui | Can't Create [%d] in Scene [%d] |
| 0x2149b4 | 0x6149b4 | .data | ascii | ui | Can't Create 3DObject [%s] in Scene [%d] |
| 0x214b80 | 0x614b80 | .data | ascii | ui | Can't Create CheckBox [%s] in Scene [%d] |
| 0x214acc | 0x614acc | .data | ascii | ui | Can't Create Edit [%s] in Scene [%d] |
| 0x214c38 | 0x614c38 | .data | ascii | ui | Can't Create Grid [%s] in Scene [%d] |
| 0x214be4 | 0x614be4 | .data | ascii | ui | Can't Create ListBox [%s] in Scene [%d] |
| 0x21496c | 0x61496c | .data | ascii | ui | Can't Create Panel [%s] in Scene [%d] |
| 0x214a0c | 0x614a0c | .data | ascii | ui | Can't Create pButton [%s] in Scene [%d] |
| 0x214b2c | 0x614b2c | .data | ascii | ui | Can't Create Progress [%s] in Scene [%d] |
| 0x214a6c | 0x614a6c | .data | ascii | ui | Can't Create Text [%s] in Scene [%d] |
| 0x2222a8 | 0x6222a8 | .data | ascii | dados | Can't generate ItemList.bin file |
| 0x223670 | 0x623670 | .data | ascii | dados | Can't generate Mixlist.bin file |
| 0x222380 | 0x622380 | .data | ascii | dados | Can't read ItemList.bin |
| 0x2236ac | 0x6236ac | .data | ascii | dados | Can't read Mixlist.bin |
| 0x2158f4 | 0x6158f4 | .data | ascii | dados | Can't read object.bin |
| 0x222348 | 0x622348 | .data | ascii | dados | Can't read SkillData.bin |
| 0x22256c | 0x62256c | .data | ascii | dados | Can't write extraitem.bin |
| 0x22273c | 0x62273c | .data | ascii | dados | Can't write inititem.bin |
| 0x21b31c | 0x61b31c | .data | ascii | dados,arquivo-de-dados | cdata.bin |
| 0x2158c4 | 0x6158c4 | .data | ascii | rede | CheckSum Error |
| 0x21b374 | 0x61b374 | .data | ascii | rede | CheckSum Error : %d,%d   m_nCheckSum=%d  nCheckSum = %d |
| 0x21b498 | 0x61b498 | .data | ascii | rede | CheckSum Error \| %d,%d  TMGround::m_nCheckSum=%d nCheckSum=%d |
| 0x1f3bea | 0x5f3bea | .text | ascii | versao | ClientPatch.dll |
| 0x222e88 | 0x622e88 | .data | ascii | dados,arquivo-de-dados | Config.bin |
| 0x215a2c | 0x615a2c | .data | ascii | ui | Create Scene Fail. |
| 0x2159c4 | 0x6159c4 | .data | ascii | dados,arquivo-de-dados | curse.bin |
| 0x222d60 | 0x622d60 | .data | ascii | dados,ui | D:\project\wyd\run\Mesh\meshTextureList.txt |
| 0x20de26 | 0x60de26 | .rdata | ascii | versao | DispatchMessageA |
| 0x216788 | 0x616788 | .data | ascii | dados,arquivo-de-dados | Effect\EffectTextureList.txt |
| 0x21526c | 0x61526c | .data | ascii | dados,arquivo-de-dados | env\character.dat |
| 0x215588 | 0x615588 | .data | ascii | dados,arquivo-de-dados | env\Character.dat |
| 0x216890 | 0x616890 | .data | ascii | dados,arquivo-de-dados | Env\EnvTextureList3.txt |
| 0x213c94 | 0x613c94 | .data | ascii | dados | env\Field%02d%02d.dat |
| 0x215144 | 0x615144 | .data | ascii | dados | Env\Field%02d%02d.dat |
| 0x2155b0 | 0x6155b0 | .data | ascii | dados,arquivo-de-dados | env\Field0813.dat |
| 0x215538 | 0x615538 | .data | ascii | dados,arquivo-de-dados | env\Field1616.dat |
| 0x21355c | 0x61355c | .data | ascii | dados,arquivo-de-dados | env\Field1817.dat |
| 0x215560 | 0x615560 | .data | ascii | dados,arquivo-de-dados | env\Field2723.dat |
| 0x215600 | 0x615600 | .data | ascii | dados,arquivo-de-dados | env\Field2922.dat |
| 0x222c94 | 0x622c94 | .data | ascii | dados,arquivo-de-dados | Env\object\mask.txt |
| 0x222554 | 0x622554 | .data | ascii | dados,arquivo-de-dados | extraitem.bin |
| 0x2131b8 | 0x6131b8 | .data | ascii | dados,arquivo-de-dados | font.txt |
| 0x1f3c4f | 0x5f3c4f | .text | ascii | versao | GamePatch.dll |
| 0x222a20 | 0x622a20 | .data | ascii | dados,arquivo-de-dados | GameRoomIP.txt |
| 0x20e67c | 0x60e67c | .rdata | ascii | versao | GetVersion |
| 0x20d84a | 0x60d84a | .rdata | ascii | versao | GetVersionExA |
| 0x213bac | 0x613bac | .data | ascii | dados,arquivo-de-dados | Guilds.txt |
| 0x2252a8 | 0x6252a8 | .data | ascii | protocolo | Header |
| 0x20e224 | 0x60e224 | .rdata | ascii | versao | ImmGetConversionStatus |
| 0x20e24e | 0x60e24e | .rdata | ascii | versao | ImmSetConversionStatus |
| 0x1fc048 | 0x5fc048 | .rdata | ascii | versao | Incompatible libpng version in application and library |
| 0x1fce44 | 0x5fce44 | .rdata | ascii | protocolo | incorrect header check |
| 0x222fc4 | 0x622fc4 | .data | ascii | ui | Initialize Interface Failed |
| 0x215a48 | 0x615a48 | .data | ascii | ui | Initialize Scene Fail. |
| 0x21027c | 0x61027c | .data | ascii | rede | Initialize single socket fail |
| 0x210244 | 0x610244 | .data | ascii | rede | Initialize socket fai |
| 0x2101dc | 0x6101dc | .data | ascii | rede | Initialize socket fail |
| 0x222ff8 | 0x622ff8 | .data | ascii | rede | Initialize Socket Failed |
| 0x222724 | 0x622724 | .data | ascii | dados,arquivo-de-dados | InitItem.bin |
| 0x222f10 | 0x622f10 | .data | ascii | dados,arquivo-de-dados | itemHelp.dat |
| 0x2159a0 | 0x6159a0 | .data | ascii | dados,arquivo-de-dados | Itemname.bin |
| 0x222138 | 0x622138 | .data | ascii | dados,arquivo-de-dados | ItemPrice.bin |
| 0x1fb2a8 | 0x5fb2a8 | .rdata | ascii | versao | JFIF APP0 marker: version %d.%02d, density %dx%d  %d |
| 0x222c50 | 0x622c50 | .data | ascii | dados,arquivo-de-dados | Mesh_MeshList.txt |
| 0x222df4 | 0x622df4 | .data | ascii | ui | Mesh\%s |
| 0x216984 | 0x616984 | .data | ascii | ui | mesh\%s.wys |
| 0x217594 | 0x617594 | .data | ascii | ui | mesh\BlackPolice.wys |
| 0x2174c0 | 0x6174c0 | .data | ascii | dados,ui,arquivo-de-dados | mesh\BlueRudol.wyt |
| 0x216f0c | 0x616f0c | .data | ascii | dados,ui,arquivo-de-dados | mesh\bm010102.wyt |
| 0x21590c | 0x61590c | .data | ascii | dados,ui,arquivo-de-dados | Mesh\BoneAni4.txt |
| 0x218118 | 0x618118 | .data | ascii | ui | mesh\ch01 |
| 0x218100 | 0x618100 | .data | ascii | ui | mesh\ch0101 |
| 0x222dc8 | 0x622dc8 | .data | ascii | ui | mesh\ch010101.wys |
| 0x222db4 | 0x622db4 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch010101.wyt |
| 0x2177dc | 0x6177dc | .data | ascii | ui | mesh\ch0101100.msh |
| 0x2177c8 | 0x6177c8 | .data | ascii | ui | mesh\ch0101100.wys |
| 0x217868 | 0x617868 | .data | ascii | ui | mesh\ch0101101.msh |
| 0x217854 | 0x617854 | .data | ascii | ui | mesh\ch0101101.wys |
| 0x217994 | 0x617994 | .data | ascii | ui | mesh\ch0101102.msh |
| 0x217980 | 0x617980 | .data | ascii | ui | mesh\ch0101102.wys |
| 0x217908 | 0x617908 | .data | ascii | ui | mesh\ch0101103.msh |
| 0x2178e0 | 0x6178e0 | .data | ascii | ui | mesh\ch0101103.wys |
| 0x217a20 | 0x617a20 | .data | ascii | ui | mesh\ch0101104.msh |
| 0x217a0c | 0x617a0c | .data | ascii | ui | mesh\ch0101104.wys |
| 0x217b38 | 0x617b38 | .data | ascii | ui | mesh\ch0101105.msh |
| 0x217b24 | 0x617b24 | .data | ascii | ui | mesh\ch0101105.wys |
| 0x217aac | 0x617aac | .data | ascii | ui | mesh\ch0101106.msh |
| 0x217a98 | 0x617a98 | .data | ascii | ui | mesh\ch0101106.wys |
| 0x217bc4 | 0x617bc4 | .data | ascii | ui | mesh\ch0101107.msh |
| 0x217bb0 | 0x617bb0 | .data | ascii | ui | mesh\ch0101107.wys |
| 0x217c50 | 0x617c50 | .data | ascii | ui | mesh\ch0101108.msh |
| 0x217c3c | 0x617c3c | .data | ascii | ui | mesh\ch0101108.wys |
| 0x217cdc | 0x617cdc | .data | ascii | ui | mesh\ch0101109.msh |
| 0x217cc8 | 0x617cc8 | .data | ascii | ui | mesh\ch0101109.wys |
| 0x217d68 | 0x617d68 | .data | ascii | ui | mesh\ch0101110.msh |
| 0x217d54 | 0x617d54 | .data | ascii | ui | mesh\ch0101110.wys |
| 0x217df4 | 0x617df4 | .data | ascii | ui | mesh\ch0101111.msh |
| 0x217de0 | 0x617de0 | .data | ascii | ui | mesh\ch0101111.wys |
| 0x217e80 | 0x617e80 | .data | ascii | ui | mesh\ch0101112.msh |
| 0x217e6c | 0x617e6c | .data | ascii | ui | mesh\ch0101112.wys |
| 0x217f0c | 0x617f0c | .data | ascii | ui | mesh\ch0101113.msh |
| 0x217ef8 | 0x617ef8 | .data | ascii | ui | mesh\ch0101113.wys |
| 0x217f98 | 0x617f98 | .data | ascii | ui | mesh\ch0101114.msh |
| 0x217f84 | 0x617f84 | .data | ascii | ui | mesh\ch0101114.wys |
| 0x2184e4 | 0x6184e4 | .data | ascii | ui | mesh\ch0101115.msh |
| 0x2184d0 | 0x6184d0 | .data | ascii | ui | mesh\ch0101115.wys |
| 0x216ea8 | 0x616ea8 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch010130.wyt |
| 0x217074 | 0x617074 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch010137.wyt |
| 0x217098 | 0x617098 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch010138.wyt |
| 0x217638 | 0x617638 | .data | ascii | ui | mesh\ch010189.msh |
| 0x218368 | 0x618368 | .data | ascii | ui | mesh\ch010195.msh |
| 0x218354 | 0x618354 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch010195.wyt |
| 0x217750 | 0x617750 | .data | ascii | ui | mesh\ch010199.msh |
| 0x21773c | 0x61773c | .data | ascii | ui | mesh\ch010199.wys |
| 0x218204 | 0x618204 | .data | ascii | ui | mesh\ch0102 |
| 0x2177f0 | 0x6177f0 | .data | ascii | ui | mesh\ch0102100.msh |
| 0x21787c | 0x61787c | .data | ascii | ui | mesh\ch0102101.msh |
| 0x2179a8 | 0x6179a8 | .data | ascii | ui | mesh\ch0102102.msh |
| 0x21791c | 0x61791c | .data | ascii | ui | mesh\ch0102103.msh |
| 0x2178f4 | 0x6178f4 | .data | ascii | ui | mesh\ch0102103.wys |
| 0x217a34 | 0x617a34 | .data | ascii | ui | mesh\ch0102104.msh |
| 0x217b4c | 0x617b4c | .data | ascii | ui | mesh\ch0102105.msh |
| 0x217ac0 | 0x617ac0 | .data | ascii | ui | mesh\ch0102106.msh |
| 0x217bd8 | 0x617bd8 | .data | ascii | ui | mesh\ch0102107.msh |
| 0x217c64 | 0x617c64 | .data | ascii | ui | mesh\ch0102108.msh |
| 0x217cf0 | 0x617cf0 | .data | ascii | ui | mesh\ch0102109.msh |
| 0x217d7c | 0x617d7c | .data | ascii | ui | mesh\ch0102110.msh |
| 0x217e08 | 0x617e08 | .data | ascii | ui | mesh\ch0102111.msh |
| 0x217e94 | 0x617e94 | .data | ascii | ui | mesh\ch0102112.msh |
| 0x217f20 | 0x617f20 | .data | ascii | ui | mesh\ch0102113.msh |
| 0x217fac | 0x617fac | .data | ascii | ui | mesh\ch0102114.msh |
| 0x2184f8 | 0x6184f8 | .data | ascii | ui | mesh\ch0102115.msh |
| 0x216da4 | 0x616da4 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch010214.wyt |
| 0x216d7c | 0x616d7c | .data | ascii | ui | mesh\ch010218.msh |
| 0x216d90 | 0x616d90 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch010219.wyt |
| 0x217064 | 0x617064 | .data | ascii | ui | mesh\ch010237 |
| 0x217088 | 0x617088 | .data | ascii | ui | mesh\ch010238 |
| 0x21764c | 0x61764c | .data | ascii | ui | mesh\ch010289.msh |
| 0x21837c | 0x61837c | .data | ascii | ui | mesh\ch010295.msh |
| 0x217764 | 0x617764 | .data | ascii | ui | mesh\ch010299.msh |
| 0x217804 | 0x617804 | .data | ascii | ui | mesh\ch0103100.msh |
| 0x217890 | 0x617890 | .data | ascii | ui | mesh\ch0103101.msh |
| 0x2179bc | 0x6179bc | .data | ascii | ui | mesh\ch0103102.msh |
| 0x217930 | 0x617930 | .data | ascii | ui | mesh\ch0103103.msh |
| 0x217a48 | 0x617a48 | .data | ascii | ui | mesh\ch0103104.msh |
| 0x217b60 | 0x617b60 | .data | ascii | ui | mesh\ch0103105.msh |
| 0x217ad4 | 0x617ad4 | .data | ascii | ui | mesh\ch0103106.msh |
| 0x217bec | 0x617bec | .data | ascii | ui | mesh\ch0103107.msh |
| 0x217c78 | 0x617c78 | .data | ascii | ui | mesh\ch0103108.msh |
| 0x217d04 | 0x617d04 | .data | ascii | ui | mesh\ch0103109.msh |
| 0x217d90 | 0x617d90 | .data | ascii | ui | mesh\ch0103110.msh |
| 0x217e1c | 0x617e1c | .data | ascii | ui | mesh\ch0103111.msh |
| 0x217ea8 | 0x617ea8 | .data | ascii | ui | mesh\ch0103112.msh |
| 0x217f34 | 0x617f34 | .data | ascii | ui | mesh\ch0103113.msh |
| 0x217fc0 | 0x617fc0 | .data | ascii | ui | mesh\ch0103114.msh |
| 0x21850c | 0x61850c | .data | ascii | ui | mesh\ch0103115.msh |
| 0x217660 | 0x617660 | .data | ascii | ui | mesh\ch010389.msh |
| 0x218390 | 0x618390 | .data | ascii | ui | mesh\ch010395.msh |
| 0x217778 | 0x617778 | .data | ascii | ui | mesh\ch010399.msh |
| 0x217818 | 0x617818 | .data | ascii | ui | mesh\ch0104100.msh |
| 0x2178a4 | 0x6178a4 | .data | ascii | ui | mesh\ch0104101.msh |
| 0x2179d0 | 0x6179d0 | .data | ascii | ui | mesh\ch0104102.msh |
| 0x217944 | 0x617944 | .data | ascii | ui | mesh\ch0104103.msh |
| 0x217a5c | 0x617a5c | .data | ascii | ui | mesh\ch0104104.msh |
| 0x217b74 | 0x617b74 | .data | ascii | ui | mesh\ch0104105.msh |
| 0x217ae8 | 0x617ae8 | .data | ascii | ui | mesh\ch0104106.msh |
| 0x217c00 | 0x617c00 | .data | ascii | ui | mesh\ch0104107.msh |
| 0x217c8c | 0x617c8c | .data | ascii | ui | mesh\ch0104108.msh |
| 0x217d18 | 0x617d18 | .data | ascii | ui | mesh\ch0104109.msh |
| 0x217da4 | 0x617da4 | .data | ascii | ui | mesh\ch0104110.msh |
| 0x217e30 | 0x617e30 | .data | ascii | ui | mesh\ch0104111.msh |
| 0x217ebc | 0x617ebc | .data | ascii | ui | mesh\ch0104112.msh |
| 0x217f48 | 0x617f48 | .data | ascii | ui | mesh\ch0104113.msh |
| 0x217fd4 | 0x617fd4 | .data | ascii | ui | mesh\ch0104114.msh |
| 0x218520 | 0x618520 | .data | ascii | ui | mesh\ch0104115.msh |
| 0x216ebc | 0x616ebc | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch010430.wyt |
| 0x217674 | 0x617674 | .data | ascii | ui | mesh\ch010489.msh |
| 0x2183a4 | 0x6183a4 | .data | ascii | ui | mesh\ch010495.msh |
| 0x21778c | 0x61778c | .data | ascii | ui | mesh\ch010499.msh |
| 0x21782c | 0x61782c | .data | ascii | ui | mesh\ch0105100.msh |
| 0x2178b8 | 0x6178b8 | .data | ascii | ui | mesh\ch0105101.msh |
| 0x2179e4 | 0x6179e4 | .data | ascii | ui | mesh\ch0105102.msh |
| 0x217958 | 0x617958 | .data | ascii | ui | mesh\ch0105103.msh |
| 0x217a70 | 0x617a70 | .data | ascii | ui | mesh\ch0105104.msh |
| 0x217b88 | 0x617b88 | .data | ascii | ui | mesh\ch0105105.msh |
| 0x217afc | 0x617afc | .data | ascii | ui | mesh\ch0105106.msh |
| 0x217c14 | 0x617c14 | .data | ascii | ui | mesh\ch0105107.msh |
| 0x217ca0 | 0x617ca0 | .data | ascii | ui | mesh\ch0105108.msh |
| 0x217d2c | 0x617d2c | .data | ascii | ui | mesh\ch0105109.msh |
| 0x217db8 | 0x617db8 | .data | ascii | ui | mesh\ch0105110.msh |
| 0x217e44 | 0x617e44 | .data | ascii | ui | mesh\ch0105111.msh |
| 0x217ed0 | 0x617ed0 | .data | ascii | ui | mesh\ch0105112.msh |
| 0x217f5c | 0x617f5c | .data | ascii | ui | mesh\ch0105113.msh |
| 0x217fe8 | 0x617fe8 | .data | ascii | ui | mesh\ch0105114.msh |
| 0x218534 | 0x618534 | .data | ascii | ui | mesh\ch0105115.msh |
| 0x216ed0 | 0x616ed0 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch010530.wyt |
| 0x217688 | 0x617688 | .data | ascii | ui | mesh\ch010589.msh |
| 0x2183b8 | 0x6183b8 | .data | ascii | ui | mesh\ch010595.msh |
| 0x2177a0 | 0x6177a0 | .data | ascii | ui | mesh\ch010599.msh |
| 0x217840 | 0x617840 | .data | ascii | ui | mesh\ch0106100.msh |
| 0x2178cc | 0x6178cc | .data | ascii | ui | mesh\ch0106101.msh |
| 0x2179f8 | 0x6179f8 | .data | ascii | ui | mesh\ch0106102.msh |
| 0x21796c | 0x61796c | .data | ascii | ui | mesh\ch0106103.msh |
| 0x217a84 | 0x617a84 | .data | ascii | ui | mesh\ch0106104.msh |
| 0x217b9c | 0x617b9c | .data | ascii | ui | mesh\ch0106105.msh |
| 0x217b10 | 0x617b10 | .data | ascii | ui | mesh\ch0106106.msh |
| 0x217c28 | 0x617c28 | .data | ascii | ui | mesh\ch0106107.msh |
| 0x217cb4 | 0x617cb4 | .data | ascii | ui | mesh\ch0106108.msh |
| 0x217d40 | 0x617d40 | .data | ascii | ui | mesh\ch0106109.msh |
| 0x217dcc | 0x617dcc | .data | ascii | ui | mesh\ch0106110.msh |
| 0x217e58 | 0x617e58 | .data | ascii | ui | mesh\ch0106111.msh |
| 0x217ee4 | 0x617ee4 | .data | ascii | ui | mesh\ch0106112.msh |
| 0x217f70 | 0x617f70 | .data | ascii | ui | mesh\ch0106113.msh |
| 0x217ffc | 0x617ffc | .data | ascii | ui | mesh\ch0106114.msh |
| 0x218548 | 0x618548 | .data | ascii | ui | mesh\ch0106115.msh |
| 0x21769c | 0x61769c | .data | ascii | ui | mesh\ch010689.msh |
| 0x2183cc | 0x6183cc | .data | ascii | ui | mesh\ch010695.msh |
| 0x2177b4 | 0x6177b4 | .data | ascii | ui | mesh\ch010699.msh |
| 0x218124 | 0x618124 | .data | ascii | ui | mesh\ch02 |
| 0x21810c | 0x61810c | .data | ascii | ui | mesh\ch0201 |
| 0x218278 | 0x618278 | .data | ascii | ui | mesh\ch020117.msh |
| 0x2170bc | 0x6170bc | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020117.wyt |
| 0x218024 | 0x618024 | .data | ascii | ui | mesh\ch020161.msh |
| 0x218010 | 0x618010 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020161.wyt |
| 0x2176c4 | 0x6176c4 | .data | ascii | ui | mesh\ch020189.msh |
| 0x218174 | 0x618174 | .data | ascii | ui | mesh\ch020190.msh |
| 0x2174ec | 0x6174ec | .data | ascii | ui | mesh\ch020194.msh |
| 0x2175ac | 0x6175ac | .data | ascii | ui | mesh\ch020195.msh |
| 0x217448 | 0x617448 | .data | ascii | ui | mesh\ch020196.msh |
| 0x2183f4 | 0x6183f4 | .data | ascii | ui | mesh\ch020197.msh |
| 0x2183e0 | 0x6183e0 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020197.wyt |
| 0x2170ac | 0x6170ac | .data | ascii | ui | mesh\ch020217 |
| 0x2182a0 | 0x6182a0 | .data | ascii | ui | mesh\ch020217.msh |
| 0x21804c | 0x61804c | .data | ascii | ui | mesh\ch020261.msh |
| 0x218038 | 0x618038 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020261.wyt |
| 0x2176d8 | 0x6176d8 | .data | ascii | ui | mesh\ch020289.msh |
| 0x218188 | 0x618188 | .data | ascii | ui | mesh\ch020290.msh |
| 0x217500 | 0x617500 | .data | ascii | ui | mesh\ch020294.msh |
| 0x2175c0 | 0x6175c0 | .data | ascii | ui | mesh\ch020295.msh |
| 0x21745c | 0x61745c | .data | ascii | ui | mesh\ch020296.msh |
| 0x21841c | 0x61841c | .data | ascii | ui | mesh\ch020297.msh |
| 0x218408 | 0x618408 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020297.wyt |
| 0x216ef8 | 0x616ef8 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020314.wyt |
| 0x216ee4 | 0x616ee4 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020315.wyt |
| 0x2182c8 | 0x6182c8 | .data | ascii | ui | mesh\ch020317.msh |
| 0x2182b4 | 0x6182b4 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020317.wyt |
| 0x218074 | 0x618074 | .data | ascii | ui | mesh\ch020357.msh |
| 0x218060 | 0x618060 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020357.wyt |
| 0x2176ec | 0x6176ec | .data | ascii | ui | mesh\ch020389.msh |
| 0x21819c | 0x61819c | .data | ascii | ui | mesh\ch020390.msh |
| 0x217514 | 0x617514 | .data | ascii | ui | mesh\ch020394.msh |
| 0x2175d4 | 0x6175d4 | .data | ascii | ui | mesh\ch020395.msh |
| 0x217470 | 0x617470 | .data | ascii | ui | mesh\ch020396.msh |
| 0x218444 | 0x618444 | .data | ascii | ui | mesh\ch020397.msh |
| 0x218430 | 0x618430 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020397.wyt |
| 0x2182f0 | 0x6182f0 | .data | ascii | ui | mesh\ch020417.msh |
| 0x2182dc | 0x6182dc | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020417.wyt |
| 0x21809c | 0x61809c | .data | ascii | ui | mesh\ch020457.msh |
| 0x218088 | 0x618088 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020457.wyt |
| 0x217700 | 0x617700 | .data | ascii | ui | mesh\ch020489.msh |
| 0x2181b0 | 0x6181b0 | .data | ascii | ui | mesh\ch020490.msh |
| 0x217528 | 0x617528 | .data | ascii | ui | mesh\ch020494.msh |
| 0x2175e8 | 0x6175e8 | .data | ascii | ui | mesh\ch020495.msh |
| 0x217484 | 0x617484 | .data | ascii | ui | mesh\ch020496.msh |
| 0x21846c | 0x61846c | .data | ascii | ui | mesh\ch020497.msh |
| 0x218458 | 0x618458 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020497.wyt |
| 0x218318 | 0x618318 | .data | ascii | ui | mesh\ch020517.msh |
| 0x218304 | 0x618304 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020517.wyt |
| 0x2180c4 | 0x6180c4 | .data | ascii | ui | mesh\ch020557.msh |
| 0x2180b0 | 0x6180b0 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020557.wyt |
| 0x217714 | 0x617714 | .data | ascii | ui | mesh\ch020589.msh |
| 0x2181c4 | 0x6181c4 | .data | ascii | ui | mesh\ch020590.msh |
| 0x21753c | 0x61753c | .data | ascii | ui | mesh\ch020594.msh |
| 0x2175fc | 0x6175fc | .data | ascii | ui | mesh\ch020595.msh |
| 0x217498 | 0x617498 | .data | ascii | ui | mesh\ch020596.msh |
| 0x218494 | 0x618494 | .data | ascii | ui | mesh\ch020597.msh |
| 0x218480 | 0x618480 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020597.wyt |
| 0x218340 | 0x618340 | .data | ascii | ui | mesh\ch020617.msh |
| 0x21832c | 0x61832c | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020617.wyt |
| 0x2180ec | 0x6180ec | .data | ascii | ui | mesh\ch020657.msh |
| 0x2180d8 | 0x6180d8 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020657.wyt |
| 0x217728 | 0x617728 | .data | ascii | ui | mesh\ch020689.msh |
| 0x2181d8 | 0x6181d8 | .data | ascii | ui | mesh\ch020690.msh |
| 0x217550 | 0x617550 | .data | ascii | ui | mesh\ch020694.msh |
| 0x217610 | 0x617610 | .data | ascii | ui | mesh\ch020695.msh |
| 0x2174ac | 0x6174ac | .data | ascii | ui | mesh\ch020696.msh |
| 0x2184bc | 0x6184bc | .data | ascii | ui | mesh\ch020697.msh |
| 0x2184a8 | 0x6184a8 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ch020697.wyt |
| 0x2176b0 | 0x6176b0 | .data | ascii | ui | mesh\DeathCos.wys |
| 0x217624 | 0x617624 | .data | ascii | ui | mesh\DeathCos2.wys |
| 0x222e74 | 0x622e74 | .data | ascii | dados,ui,arquivo-de-dados | Mesh\ed.dat |
| 0x21757c | 0x61757c | .data | ascii | ui | mesh\GreenPolice.wys |
| 0x2170d0 | 0x6170d0 | .data | ascii | dados,ui,arquivo-de-dados | mesh\LB010101.wyt |
| 0x2170e4 | 0x6170e4 | .data | ascii | dados,ui,arquivo-de-dados | mesh\LB010201.wyt |
| 0x2170f8 | 0x6170f8 | .data | ascii | dados,ui,arquivo-de-dados | mesh\LB010301.wyt |
| 0x21710c | 0x61710c | .data | ascii | dados,ui,arquivo-de-dados | mesh\LK010101.wyt |
| 0x217120 | 0x617120 | .data | ascii | dados,ui,arquivo-de-dados | mesh\LK010201.wyt |
| 0x217134 | 0x617134 | .data | ascii | dados,ui,arquivo-de-dados | mesh\LK010301.wyt |
| 0x2157e8 | 0x6157e8 | .data | ascii | dados,ui,arquivo-de-dados | mesh\MeshList.txt |
| 0x216810 | 0x616810 | .data | ascii | dados,ui,arquivo-de-dados | Mesh\MeshTextureList.txt |
| 0x216f20 | 0x616f20 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mi010105.wyt |
| 0x2171b4 | 0x6171b4 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101170.wyt |
| 0x2171c8 | 0x6171c8 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101171.wyt |
| 0x2171dc | 0x6171dc | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101172.wyt |
| 0x2171f0 | 0x6171f0 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101173.wyt |
| 0x217204 | 0x617204 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101174.wyt |
| 0x217218 | 0x617218 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101175.wyt |
| 0x21722c | 0x61722c | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101176.wyt |
| 0x217240 | 0x617240 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101177.wyt |
| 0x217254 | 0x617254 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101178.wyt |
| 0x217268 | 0x617268 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101179.wyt |
| 0x21727c | 0x61727c | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101180.wyt |
| 0x217290 | 0x617290 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101181.wyt |
| 0x2172a4 | 0x6172a4 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101182.wyt |
| 0x2172b8 | 0x6172b8 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101183.wyt |
| 0x2172cc | 0x6172cc | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101184.wyt |
| 0x2172e0 | 0x6172e0 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101185.wyt |
| 0x2172f4 | 0x6172f4 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101186.wyt |
| 0x217308 | 0x617308 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101187.wyt |
| 0x21731c | 0x61731c | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101188.wyt |
| 0x217330 | 0x617330 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101189.wyt |
| 0x217344 | 0x617344 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101190.wyt |
| 0x217358 | 0x617358 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101191.wyt |
| 0x21736c | 0x61736c | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101192.wyt |
| 0x217380 | 0x617380 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101193.wyt |
| 0x217394 | 0x617394 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101194.wyt |
| 0x2173a8 | 0x6173a8 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101195.wyt |
| 0x2173bc | 0x6173bc | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101196.wyt |
| 0x2173d0 | 0x6173d0 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101197.wyt |
| 0x2173e4 | 0x6173e4 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101198.wyt |
| 0x2173f8 | 0x6173f8 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101199.wyt |
| 0x21740c | 0x61740c | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt0101200.wyt |
| 0x216de0 | 0x616de0 | .data | ascii | ui | mesh\mt010124.msh |
| 0x216dcc | 0x616dcc | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt010124.wyt |
| 0x216db8 | 0x616db8 | .data | ascii | ui | mesh\mt010131.msh |
| 0x216df4 | 0x616df4 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt010132.wyt |
| 0x216e08 | 0x616e08 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt010133.wyt |
| 0x216e1c | 0x616e1c | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt010134.wyt |
| 0x216e30 | 0x616e30 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt010135.wyt |
| 0x216e44 | 0x616e44 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt010136.wyt |
| 0x216e58 | 0x616e58 | .data | ascii | dados,ui,arquivo-de-dados | mesh\mt010137.wyt |
| 0x21855c | 0x61855c | .data | ascii | ui | mesh\newmt.msh |
| 0x21856c | 0x61856c | .data | ascii | ui | mesh\newmtB000.wys |
| 0x218580 | 0x618580 | .data | ascii | ui | mesh\newmtR000.wys |
| 0x218594 | 0x618594 | .data | ascii | ui | mesh\newmtW000.wys |
| 0x2174d4 | 0x6174d4 | .data | ascii | dados,ui,arquivo-de-dados | mesh\PurpleRudol.wyt |
| 0x217434 | 0x617434 | .data | ascii | dados,ui,arquivo-de-dados | mesh\RedSanta.wyt |
| 0x218160 | 0x618160 | .data | ascii | dados,ui,arquivo-de-dados | mesh\SpiderCos.wyt |
| 0x2232b0 | 0x6232b0 | .data | ascii | ui | mesh\stone01.wys |
| 0x222e68 | 0x622e68 | .data | ascii | dados,ui,arquivo-de-dados | Mesh\tn.dat |
| 0x216f34 | 0x616f34 | .data | ascii | ui | mesh\tr13 |
| 0x216f40 | 0x616f40 | .data | ascii | dados,ui,arquivo-de-dados | mesh\tr130101.wyt |
| 0x216f54 | 0x616f54 | .data | ascii | ui | mesh\tr14 |
| 0x216f74 | 0x616f74 | .data | ascii | ui | mesh\tr15 |
| 0x216f94 | 0x616f94 | .data | ascii | ui | mesh\tr16 |
| 0x216fb4 | 0x616fb4 | .data | ascii | ui | mesh\tr17 |
| 0x216fe4 | 0x616fe4 | .data | ascii | dados,ui,arquivo-de-dados | mesh\tr180101.wyt |
| 0x217008 | 0x617008 | .data | ascii | dados,ui,arquivo-de-dados | mesh\tr180102.wyt |
| 0x216fd4 | 0x616fd4 | .data | ascii | ui | mesh\tr190101 |
| 0x216ff8 | 0x616ff8 | .data | ascii | ui | mesh\tr190102 |
| 0x21701c | 0x61701c | .data | ascii | ui | mesh\tr200101 |
| 0x217040 | 0x617040 | .data | ascii | ui | mesh\tr200102 |
| 0x215928 | 0x615928 | .data | ascii | dados,ui,arquivo-de-dados | mesh\ValidIndex.bin |
| 0x217564 | 0x617564 | .data | ascii | ui | mesh\WhitePolice.wys |
| 0x217420 | 0x617420 | .data | ascii | dados,ui,arquivo-de-dados | mesh\WhiteSanta.wyt |
| 0x222d94 | 0x622d94 | .data | ascii | dados,arquivo-de-dados | MeshTextureList.txt |
| 0x213b80 | 0x613b80 | .data | ascii | dados,arquivo-de-dados | minimap.dat |
| 0x222920 | 0x622920 | .data | ascii | dados,arquivo-de-dados | Missionitems.bin |
| 0x222f40 | 0x622f40 | .data | ascii | dados,arquivo-de-dados | Mixhelp.dat |
| 0x222e5c | 0x622e5c | .data | ascii | dados,arquivo-de-dados | Music.txt |
| 0x2119e8 | 0x6119e8 | .data | ascii | ui | music\CastleWar.mp3 |
| 0x2114e8 | 0x6114e8 | .data | ascii | ui | music\dungeon01.mp3 |
| 0x2116e8 | 0x6116e8 | .data | ascii | ui | music\dungeon02.mp3 |
| 0x2111e8 | 0x6111e8 | .data | ascii | ui | music\field01.mp3 |
| 0x2113e8 | 0x6113e8 | .data | ascii | ui | music\field02.mp3 |
| 0x2118e8 | 0x6118e8 | .data | ascii | ui | music\field03.mp3 |
| 0x211ae8 | 0x611ae8 | .data | ascii | ui | music\kepra.mp3 |
| 0x211be8 | 0x611be8 | .data | ascii | ui | music\khepraBoss.mp3 |
| 0x2115e8 | 0x6115e8 | .data | ascii | ui | music\kingdom.mp3 |
| 0x210fe8 | 0x610fe8 | .data | ascii | ui | music\login.mp3 |
| 0x2110e8 | 0x6110e8 | .data | ascii | ui | music\town01.mp3 |
| 0x2112e8 | 0x6112e8 | .data | ascii | ui | music\town02.mp3 |
| 0x2117e8 | 0x6117e8 | .data | ascii | ui | music\town03.mp3 |
| 0x213cc8 | 0x613cc8 | .data | ascii | dados,arquivo-de-dados | notice.txt |
| 0x2158b8 | 0x6158b8 | .data | ascii | dados,arquivo-de-dados | object.bin |
| 0x2158d4 | 0x6158d4 | .data | ascii | dados | Object.bin File Crashed |
| 0x1fc0b8 | 0x5fc0b8 | .rdata | ascii | versao | PNG file corrupted by ASCII conversion |
| 0x2132e0 | 0x6132e0 | .data | ascii | dados,arquivo-de-dados | RenderStateLog.txt |
| 0x22243c | 0x62243c | .data | ascii | dados | S:/Settings/Lotto.txt |
| 0x21330c | 0x61330c | .data | ascii | dados,arquivo-de-dados | SamplerStateLog.txt |
| 0x213274 | 0x613274 | .data | ascii | dados | Shader\pseffect%d.bin |
| 0x213204 | 0x613204 | .data | ascii | dados | Shader\skinmesh%d.bin |
| 0x21323c | 0x61323c | .data | ascii | dados | Shader\vseffect%d.bin |
| 0x22224c | 0x62224c | .data | ascii | dados,arquivo-de-dados | SkillData.bin |
| 0x2230c4 | 0x6230c4 | .data | ascii | dados,rede,arquivo-de-dados | sn.bin |
| 0x212ecc | 0x612ecc | .data | ascii | dados,ui,arquivo-de-dados | sound\soundlist.txt |
| 0x213340 | 0x613340 | .data | ascii | dados,arquivo-de-dados | TextureStageStateLog.txt |
| 0x222c7c | 0x622c7c | .data | ascii | dados,arquivo-de-dados | ui_hellStoredesc.txt |
| 0x222c64 | 0x622c64 | .data | ascii | dados,arquivo-de-dados | ui_UITextureSetList.txt |
| 0x222224 | 0x622224 | .data | ascii | dados,arquivo-de-dados | UI/strdef.bin |
| 0x215440 | 0x615440 | .data | ascii | dados,arquivo-de-dados | UI\chardescbeast.txt |
| 0x215428 | 0x615428 | .data | ascii | dados,arquivo-de-dados | UI\chardescfoema.txt |
| 0x215458 | 0x615458 | .data | ascii | dados,arquivo-de-dados | UI\chardeschunter.txt |
| 0x215410 | 0x615410 | .data | ascii | dados,arquivo-de-dados | UI\chardesctrans.txt |
| 0x213d20 | 0x613d20 | .data | ascii | dados,arquivo-de-dados | UI\command.txt |
| 0x215650 | 0x615650 | .data | ascii | dados,arquivo-de-dados | UI\demo.bin |
| 0x215660 | 0x615660 | .data | ascii | dados,arquivo-de-dados | UI\demo2.bin |
| 0x215674 | 0x615674 | .data | ascii | dados,arquivo-de-dados | UI\demo3.bin |
| 0x215688 | 0x615688 | .data | ascii | dados,arquivo-de-dados | UI\demo4.bin |
| 0x222cf4 | 0x622cf4 | .data | ascii | dados,arquivo-de-dados | UI\EffectString.txt |
| 0x222d10 | 0x622d10 | .data | ascii | dados,arquivo-de-dados | UI\EffectSubString.txt |
| 0x2135ac | 0x6135ac | .data | ascii | dados,arquivo-de-dados | UI\EndDemo.bin |
| 0x213520 | 0x613520 | .data | ascii | dados,arquivo-de-dados | UI\Ending.bin |
| 0x213d30 | 0x613d30 | .data | ascii | dados,arquivo-de-dados | UI\etc.txt |
| 0x213bf8 | 0x613bf8 | .data | ascii | dados,ui,arquivo-de-dados | UI\FieldScene2.txt |
| 0x222d30 | 0x622d30 | .data | ascii | dados,arquivo-de-dados | UI\GuildString.txt |
| 0x222ca8 | 0x622ca8 | .data | ascii | dados,arquivo-de-dados | UI\hellgate\icon.wyt |
| 0x213f5c | 0x613f5c | .data | ascii | dados,arquivo-de-dados | UI\hellStoredesc.txt |
| 0x213d0c | 0x613d0c | .data | ascii | dados,ui,arquivo-de-dados | UI\interface.txt |
| 0x213d3c | 0x613d3c | .data | ascii | dados,ui,arquivo-de-dados | UI\interface1.txt |
| 0x213d50 | 0x613d50 | .data | ascii | dados,ui,arquivo-de-dados | UI\interface2.txt |
| 0x213d64 | 0x613d64 | .data | ascii | dados,ui,arquivo-de-dados | UI\interface3.txt |
| 0x21b3d0 | 0x61b3d0 | .data | ascii | dados | UI\m%02d%02d.wyt |
| 0x20f9b0 | 0x60f9b0 | .data | ascii | dados,arquivo-de-dados | UI\minimap.wyt |
| 0x213c6c | 0x613c6c | .data | ascii | dados,arquivo-de-dados | UI\mix4desc.txt |
| 0x213f74 | 0x613f74 | .data | ascii | dados,arquivo-de-dados | UI\PotalPos.txt |
| 0x214508 | 0x614508 | .data | ascii | dados,arquivo-de-dados | UI\QuestContents.txt |
| 0x214520 | 0x614520 | .data | ascii | dados,arquivo-de-dados | UI\QuestContents2.txt |
| 0x214538 | 0x614538 | .data | ascii | dados,arquivo-de-dados | UI\QuestContents3.txt |
| 0x214550 | 0x614550 | .data | ascii | dados,arquivo-de-dados | UI\QuestContents4.txt |
| 0x213dd8 | 0x613dd8 | .data | ascii | dados,arquivo-de-dados | UI\QuestMessage.txt |
| 0x213d78 | 0x613d78 | .data | ascii | dados,arquivo-de-dados | UI\QuestSubjects.txt |
| 0x213d90 | 0x613d90 | .data | ascii | dados,arquivo-de-dados | UI\QuestSubjects2.txt |
| 0x213da8 | 0x613da8 | .data | ascii | dados,arquivo-de-dados | UI\QuestSubjects3.txt |
| 0x213dc0 | 0x613dc0 | .data | ascii | dados,arquivo-de-dados | UI\QuestSubjects4.txt |
| 0x2159b4 | 0x6159b4 | .data | ascii | dados,arquivo-de-dados | UI\RC.bin |
| 0x2152cc | 0x6152cc | .data | ascii | dados,arquivo-de-dados | UI\selchar.txt |
| 0x215240 | 0x615240 | .data | ascii | dados,ui,arquivo-de-dados | UI\SelCharScene2.txt |
| 0x2154e4 | 0x6154e4 | .data | ascii | dados,ui,arquivo-de-dados | UI\SelServerScene2.txt |
| 0x2135cc | 0x6135cc | .data | ascii | dados,arquivo-de-dados | UI\TimeTable.bin |
| 0x222d48 | 0x622d48 | .data | ascii | dados,arquivo-de-dados | UI\UIString.txt |
| 0x216694 | 0x616694 | .data | ascii | dados,arquivo-de-dados | UI\UITextureListN.txt |
| 0x21670c | 0x61670c | .data | ascii | dados,arquivo-de-dados | UI\UITextureSetList.txt |
| 0x1fbb40 | 0x5fbb40 | .rdata | ascii | versao | Unsupported color conversion request |
| 0x20de16 | 0x60de16 | .rdata | ascii | versao | UpdateWindow |
| 0x210553 | 0x610553 | .data | ascii | versao | want to check that the latest version of |
| 0x2105cd | 0x6105cd | .data | ascii | protocolo | was compiled with header files that match |
| 0x1fbdb4 | 0x5fbdb4 | .rdata | ascii | versao | Wrong JPEG library version: library is %d, caller expects %d |
| 0x222e38 | 0x622e38 | .data | ascii | dados | wyd.log |
| 0x222cd4 | 0x622cd4 | .data | ascii | dados | WYD.log |
| 0x231148 | 0x631148 | .data | ascii | protecao | XTrapVa.dll |
| 0x1fc004 | 0x5fc004 | .rdata | ascii | versao | zlib version error |

## Como regenerar

```bash
node apps/tools/bin/wyd-tools.mjs strings client --bin WYD.exe
```

