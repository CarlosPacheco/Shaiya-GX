// Prints immediates stored into [reg+0xc] and [reg+0x10] (frame width/height for
// the drag helper 0x526f30) per function, with referenced texture names.
import ghidra.app.script.GhidraScript;
import ghidra.program.model.listing.*;
import ghidra.program.model.data.*;
import ghidra.program.model.scalar.Scalar;
public class DragRectsImm extends GhidraScript {
  @Override public void run() throws Exception {
    FunctionIterator it = currentProgram.getFunctionManager().getFunctions(true);
    while (it.hasNext()) {
      Function f = it.next();
      InstructionIterator ii = currentProgram.getListing().getInstructions(f.getBody(), true);
      Long w=null,h=null; StringBuilder tex=new StringBuilder();
      while (ii.hasNext()) {
        Instruction in = ii.next();
        if (in.getMnemonicString().equals("MOV") && in.getNumOperands()==2) {
          String d = in.getDefaultOperandRepresentation(0);
          Object[] o = in.getOpObjects(1);
          if (o.length==1 && o[0] instanceof Scalar && d.contains("[") ) {
            long v=((Scalar)o[0]).getSignedValue();
            if (d.matches(".*\\+ 0xc\\]")) w=v; else if (d.matches(".*\\+ 0x10\\]")) h=v;
          }
        }
        for (int k=0;k<in.getNumOperands();k++) for (Object o : in.getOpObjects(k)) if (o instanceof Scalar) {
          long v=((Scalar)o).getUnsignedValue();
          if (v>=0x6b0000&&v<0x6d0000){Data dd=getDataAt(toAddr(v)); if(dd!=null&&dd.getValue() instanceof String){String s=(String)dd.getValue(); if(s.toLowerCase().endsWith(".tga")&&!tex.toString().contains(s)) tex.append(s).append(' ');}}
        }
      }
      if (w!=null&&h!=null&&w>8&&w<1100&&h>8&&h<800) println(f.getEntryPoint()+" w="+w+" h="+h+" "+tex);
    }
  }
}
