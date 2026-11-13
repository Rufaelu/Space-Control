package main

import "game/internal/gl"

// All GLSL lives here. It is written for OpenGL 3.3 core and also compiles as
// OpenGL ES 3.0 (Android): only the header differs.

const glslHead = gl.GLSLHead

const glslNoise = `
float hash(vec3 p){ p=fract(p*0.3183099+0.1); p*=17.0; return fract(p.x*p.y*p.z*(p.x+p.y+p.z)); }
float noise(vec3 x){
    vec3 i=floor(x), f=fract(x); f=f*f*(3.0-2.0*f);
    return mix(mix(mix(hash(i),hash(i+vec3(1,0,0)),f.x), mix(hash(i+vec3(0,1,0)),hash(i+vec3(1,1,0)),f.x),f.y),
               mix(mix(hash(i+vec3(0,0,1)),hash(i+vec3(1,0,1)),f.x), mix(hash(i+vec3(0,1,1)),hash(i+vec3(1,1,1)),f.x),f.y),f.z);
}
float fbm(vec3 p){ float a=0.5,s=0.0; for(int i=0;i<6;i++){ s+=a*noise(p); p=p*2.02+vec3(1.7,9.2,3.1); a*=0.5; } return s; }
`

// Fullscreen triangle, no vertex buffer needed.
const fsTriVS = glslHead + `
out vec2 vUV;
void main(){
    vec2 p=vec2((gl_VertexID<<1)&2, gl_VertexID&2);
    vUV=p; gl_Position=vec4(p*2.0-1.0,0.0,1.0);
}` + "\x00"

// Bakes the nebula, faint star dust and the sun into one cubemap face.
const bakeFS = glslHead + glslNoise + `
in vec2 vUV; out vec4 o;
uniform vec3 uF,uR,uU,uSun; uniform float uSeed;
void main(){
    vec2 sc=vUV*2.0-1.0;
    vec3 d=normalize(uF+sc.x*uR+sc.y*uU);
    vec3 p=d*2.2+uSeed;
    vec3 w=vec3(fbm(p),fbm(p+vec3(5.2,1.3,2.8)),fbm(p+vec3(1.7,9.2,4.1)));
    float n1=fbm(p+2.0*w);
    float n2=fbm(p*1.7-1.5*w+7.3);
    float band=exp(-pow(dot(d,normalize(vec3(0.35,0.85,0.2)))*2.2,2.0));
    float dens=smoothstep(0.40,0.80,n1)*(0.30+0.70*band);
    float dens2=smoothstep(0.45,0.85,n2)*(0.20+0.80*band);
    vec3 c1=mix(vec3(0.04,0.10,0.50),vec3(0.55,0.08,0.60),clamp(w.x*1.5-0.2,0.0,1.0));
    vec3 c2=mix(vec3(0.0,0.45,0.55),vec3(1.0,0.32,0.12),clamp(w.y*1.5-0.2,0.0,1.0));
    vec3 col=vec3(0.004,0.006,0.016);
    col+=c1*dens*0.55+c2*dens2*dens2*0.6;
    col+=vec3(1.0,0.75,0.9)*pow(dens,4.0)*0.3;
    float dust=smoothstep(0.45,0.75,fbm(p*3.1+w*2.0+11.0));
    col*=1.0-0.65*dust*band;
    vec3 g=d*230.0; vec3 ci=floor(g); vec3 cf=fract(g)-0.5;
    float h=hash(ci);
    float st=step(0.984,h)*smoothstep(0.40,0.0,length(cf));
    col+=vec3(0.8,0.86,1.0)*st*(0.25+1.3*band)*fract(h*91.7);
    float sd=max(dot(d,uSun),0.0);
    col+=vec3(1.0,0.80,0.58)*(pow(sd,2500.0)*50.0+pow(sd,180.0)*1.2+pow(sd,10.0)*0.22);
    o=vec4(col,1.0);
}` + "\x00"

const skyFS = glslHead + `
in vec2 vUV; out vec4 o;
uniform samplerCube uSky; uniform vec3 uF,uR,uU; uniform vec2 uTan;
void main(){
    vec2 sc=vUV*2.0-1.0;
    vec3 d=normalize(uF+sc.x*uTan.x*uR+sc.y*uTan.y*uU);
    o=vec4(texture(uSky,d).rgb,1.0);
}` + "\x00"

const starVS = glslHead + `
layout(location=0) in vec3 aDir;
layout(location=1) in vec4 aCol;
layout(location=2) in float aPhase;
uniform mat4 uVP; uniform float uTime,uScale;
out vec3 vCol;
void main(){
    vec4 p=uVP*vec4(aDir,0.0);
    gl_Position=p.xyww;
    float tw=0.72+0.28*sin(uTime*(1.2+aPhase*3.0)+aPhase*40.0);
    gl_PointSize=max(aCol.a*uScale,1.0);
    vCol=aCol.rgb*tw;
}` + "\x00"

const starFS = glslHead + `
in vec3 vCol; out vec4 o;
void main(){
    vec2 c=gl_PointCoord*2.0-1.0;
    o=vec4(vCol*exp(-dot(c,c)*3.5),1.0);
}` + "\x00"

const meshVS = glslHead + `
layout(location=0) in vec3 aPos;
layout(location=1) in vec3 aN;
layout(location=2) in vec4 aCol;
uniform mat4 uVP,uModel;
out vec3 vN,vW; out vec4 vC;
void main(){
    vec4 w=uModel*vec4(aPos,1.0);
    vW=w.xyz; vN=mat3(uModel)*aN; vC=aCol;
    gl_Position=uVP*w;
}` + "\x00"

const meshFS = glslHead + `
in vec3 vN,vW; in vec4 vC; out vec4 o;
uniform vec3 uSun,uCam,uTint; uniform vec4 uRim;
uniform float uFlash,uEmis,uSpec,uAmb;
void main(){
    vec3 n=normalize(vN), v=normalize(uCam-vW);
    vec3 sunCol=vec3(1.0,0.90,0.78);
    float diff=max(dot(n,uSun),0.0);
    vec3 amb=mix(vec3(0.05,0.07,0.16),vec3(0.20,0.14,0.28),n.y*0.5+0.5)*uAmb;
    float rim=pow(1.0-max(dot(n,v),0.0),3.0);
    float spec=pow(max(dot(n,normalize(uSun+v)),0.0),40.0)*uSpec;
    vec3 base=vC.rgb*uTint;
    vec3 col=base*(amb+diff*sunCol*1.25)+spec*sunCol+rim*uRim.rgb*uRim.a*(0.25+0.75*diff);
    col=mix(col,base*uEmis,vC.a);
    col+=vec3(uFlash);
    o=vec4(col,1.0);
}` + "\x00"

// Additive billboards: shape 0 = soft glow, 1 = ring, 2 = beam.
const partVS = glslHead + `
layout(location=0) in vec3 aPos;
layout(location=1) in vec3 aUV;
layout(location=2) in vec4 aCol;
uniform mat4 uVP;
out vec2 vUV; flat out float vShape; out vec4 vCol;
void main(){ vUV=aUV.xy; vShape=aUV.z; vCol=aCol; gl_Position=uVP*vec4(aPos,1.0); }` + "\x00"

const partFS = glslHead + `
in vec2 vUV; flat in float vShape; in vec4 vCol; out vec4 o;
void main(){
    float a;
    if(vShape<0.5){ float r=length(vUV); a=pow(max(1.0-r,0.0),2.0); }
    else if(vShape<1.5){ float r=length(vUV); float q=(r-0.82)/0.07; a=exp(-q*q); }
    else { a=pow(max(1.0-abs(vUV.y),0.0),2.5)*smoothstep(1.0,0.7,abs(vUV.x)); }
    o=vec4(vCol.rgb*vCol.a*a,1.0);
}` + "\x00"

// Bright pass + 4-tap downsample. uThresh<=0 means plain downsample.
const brightFS = glslHead + `
in vec2 vUV; out vec4 o;
uniform sampler2D uTex; uniform vec2 uTexel; uniform float uThresh;
void main(){
    vec3 c=texture(uTex,vUV+uTexel*vec2(-1,-1)).rgb+texture(uTex,vUV+uTexel*vec2(1,-1)).rgb
          +texture(uTex,vUV+uTexel*vec2(-1,1)).rgb+texture(uTex,vUV+uTexel*vec2(1,1)).rgb;
    c*=0.25;
    c=min(c,vec3(24.0));
    if(uThresh>0.0){ float l=max(c.r,max(c.g,c.b)); c*=smoothstep(uThresh,uThresh+0.7,l); }
    o=vec4(c,1.0);
}` + "\x00"

const blurFS = glslHead + `
in vec2 vUV; out vec4 o;
uniform sampler2D uTex; uniform vec2 uDir;
void main(){
    vec3 c=texture(uTex,vUV).rgb*0.2270270270;
    c+=(texture(uTex,vUV+uDir*1.3846153846).rgb+texture(uTex,vUV-uDir*1.3846153846).rgb)*0.3162162162;
    c+=(texture(uTex,vUV+uDir*3.2307692308).rgb+texture(uTex,vUV-uDir*3.2307692308).rgb)*0.0702702703;
    o=vec4(c,1.0);
}` + "\x00"

const compositeFS = glslHead + `
in vec2 vUV; out vec4 o;
uniform sampler2D uScene,uB1,uB2;
uniform float uTime,uAb,uHit,uFade;
uniform vec3 uHitCol;
float h21(vec2 p){ return fract(sin(dot(p,vec2(12.9898,78.233)))*43758.5453); }
vec3 aces(vec3 x){ return clamp((x*(2.51*x+0.03))/(x*(2.43*x+0.59)+0.14),0.0,1.0); }
void main(){
    vec2 d=vUV-0.5; float r=length(d);
    vec2 off=d*r*uAb;
    vec3 c=vec3(texture(uScene,vUV+off).r,texture(uScene,vUV).g,texture(uScene,vUV-off).b);
    c+=texture(uB1,vUV).rgb*0.50+texture(uB2,vUV).rgb*0.70;
    c*=mix(1.0,0.45,smoothstep(0.30,0.95,r));
    c+=uHitCol*uHit*smoothstep(0.10,0.75,r);
    c=aces(c);
    c=pow(c,vec3(1.0/2.2));
    c+=(h21(gl_FragCoord.xy+fract(uTime)*91.0)-0.5)*0.018;
    o=vec4(c*uFade,1.0);
}` + "\x00"

const hudVS = glslHead + `
layout(location=0) in vec2 aPos;
layout(location=1) in vec4 aCol;
uniform vec2 uRes; out vec4 vCol;
void main(){ vCol=aCol; gl_Position=vec4(aPos.x/uRes.x*2.0-1.0,1.0-aPos.y/uRes.y*2.0,0.0,1.0); }` + "\x00"

const hudFS = glslHead + `
in vec4 vCol; out vec4 o;
void main(){ o=vCol; }` + "\x00"
