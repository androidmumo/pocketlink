/* Capture is enabled only after a current server grant. No audio is retained. */
class PocketCapture extends AudioWorkletProcessor {
 constructor(){super();this.stream=0;this.outstanding=0;this.phase=0;this.sum=0;this.count=0;this.index=0;this.frame=new Int16Array(320);this.port.onmessage=e=>{if(e.data==='ack'){this.outstanding=Math.max(0,this.outstanding-1);return;}this.stream=e.data?.stream||0;this.phase=this.sum=this.count=this.index=0;};}
 process(inputs){
  const input=inputs[0]?.[0];if(!this.stream||!input)return true;
  for(const sample of input){this.sum+=sample;this.count++;this.phase+=16000;if(this.phase>=sampleRate){this.phase-=sampleRate;const value=Math.max(-1,Math.min(1,this.sum/this.count));this.frame[this.index++]=Math.round(value*(value<0?32768:32767));this.sum=this.count=0;if(this.index===320){if(this.outstanding<8){this.outstanding++;this.port.postMessage({stream:this.stream,pcm:this.frame.buffer},[this.frame.buffer]);this.frame=new Int16Array(320);}this.index=0;}}}
  return true;
 }
}
registerProcessor('pocketlink-capture',PocketCapture);
