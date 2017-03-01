package example;

import net.sf.json.JSONArray;
import net.sf.json.JSONObject;
import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import org.hyperledger.fabric.sdk.shim.ChaincodeBase;
import org.hyperledger.fabric.sdk.shim.ChaincodeStub;
import org.hyperledger.fabric.sdk.security.CryptoPrimitives;
import org.hyperledger.protos.TableProto;
import sun.misc.BASE64Decoder;

import java.util.ArrayList;
import java.util.List;

/**
 * Created by zerppen on 12/26/16.
 *
 *  我选择用String代替BYTES的原因有2：
 *  1.java直接存BYTES，后续转换比较费时间
 *  2.table-proto中插入表中的数据都是先查询出来，而string在switch case中的顺序排在第一位，BYTES是第6位，更快
 */

public class KRCC extends ChaincodeBase{
    private static final String TableCurrency           = "Currency";
    private static final String TableCurrencyReleaseLog = "CurrencyReleaseLog";
    private static final String TableCurrencyAssignLog  = "CurrencyAssignLog";
    private static final String TableAssets             = "Assets";
    private static final String TableAssetLockLog       = "AssetLockLog";
    private static final String TableTxLog              = "TxLog";
    private static final String TableTxLog2             = "TxLog2";
    private static final String CNY                     = "CNY";
    private static final String USD                     = "USD";
    private static final String CheckErr                = "CheckErr"; // "-1"
    private static final String WorldStateErr           = "WdErr";    //"-2"

    private static  Log log = LogFactory.getLog(KRCC.class);

    // *******所有币数量相关数据，均由APP四舍五入保留六位小数然后乘10^6************
    // *********因为chaincode table不支持float类型，不同语言的float精度处理也不同******

    @Override
    public String run(ChaincodeStub stub, String function, String[] args) {
        log.info("In run, function:"+function);
        String ret = null;


        if((args.length)!=0){
            ret = "Incorrect number of arguments. Expecting 3,in release currency";
            return ret;
        }
        String reCT = createTable(stub);
        if(reCT!=null){
            log.error("run error1");
            return ret;
        }
        String reIT = initTable(stub);
        if(reIT!=null){
            log.error("run error2");
            return ret;
        }

        return ret;
    }

    public String createTable(ChaincodeStub stub){
        List<TableProto.ColumnDefinition> cols = new ArrayList<TableProto.ColumnDefinition>();

        String retStr = null;


        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("ID")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.STRING)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("Count")
                .setKey(false)
                .setType(TableProto.ColumnDefinition.Type.INT64)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("LeftCount")
                .setKey(false)
                .setType(TableProto.ColumnDefinition.Type.INT64)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("Creator")
                .setKey(false)
                .setType(TableProto.ColumnDefinition.Type.STRING) //go中是BYTES
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("CreateTime")
                .setKey(false)
                .setType(TableProto.ColumnDefinition.Type.INT64)
                .build()
        );
        //币种table
        try {
            stub.createTable(TableCurrency,cols);
        }catch (Exception e){
            retStr = e.toString();
            return retStr;
        }

        cols.clear();

        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("Currency")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.STRING)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("Count")
                .setKey(false)
                .setType(TableProto.ColumnDefinition.Type.INT64)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("ReleaseTime")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.INT64)
                .build()
        );
        // 币发行log table
        try {
            stub.createTable(TableCurrencyReleaseLog,cols);
        }catch (Exception e){
            retStr = e.toString();
            return retStr;
        }

        cols.clear();

        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("Currency")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.STRING)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("Owner")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.STRING)//go中为BYTES
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("Count")
                .setKey(false)
                .setType(TableProto.ColumnDefinition.Type.INT64)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("AssignTime")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.INT64)
                .build()
        );
        //币发行log table
        try {
            stub.createTable(TableCurrencyAssignLog,cols);
        }catch (Exception e){
            retStr = e.toString();
            return retStr;
        }

        cols.clear();

        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("Owner")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.STRING)//go中为BYTES
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("Currency")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.STRING)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("Count")
                .setKey(false)
                .setType(TableProto.ColumnDefinition.Type.INT64)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("LockCount")
                .setKey(false)
                .setType(TableProto.ColumnDefinition.Type.INT64)
                .build()
        );
        //账户资产信息 table
        try {
            stub.createTable(TableAssets,cols);
        }catch (Exception e){
            retStr = e.toString();
            return retStr;
        }

        cols.clear();

        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("Owner")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.STRING)//go中为BYTES
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("Currency")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.STRING)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("Order")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.STRING)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("IsLock")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.BOOL)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("LockCount")
                .setKey(false)
                .setType(TableProto.ColumnDefinition.Type.INT64)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("LockTime")
                .setKey(false)
                .setType(TableProto.ColumnDefinition.Type.INT64)
                .build()
        );
        //账户余额锁定log table
        try {
            stub.createTable(TableAssetLockLog,cols);
        }catch (Exception e){
            retStr = e.toString();
            return retStr;
        }

        cols.clear();

        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("Owner")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.STRING)//go中为BYTES
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("SrcCurrency")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.STRING)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("DesCurrency")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.STRING)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("RawOrder")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.STRING)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("Detail")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.STRING)//go中为BYTES
                .build()
        );

        //交易log table
        try {
            stub.createTable(TableTxLog,cols);
        }catch (Exception e){
            retStr = e.toString();
            return retStr;
        }

        cols.clear();
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("UUID")
                .setKey(true)
                .setType(TableProto.ColumnDefinition.Type.STRING)
                .build()
        );
        cols.add(TableProto.ColumnDefinition.newBuilder()
                .setName("Detail")
                .setKey(false)
                .setType(TableProto.ColumnDefinition.Type.STRING)//go中为BYTES
                .build()
        );
        //交易log table  冗余表便于查询
        try {
            stub.createTable(TableTxLog2,cols);
        }catch (Exception e){
            retStr = e.toString();
            return retStr;
        }

        return retStr;
    }

    public String initTable(ChaincodeStub stub){

        String ret = null;
        //内置人民币CNY
        TableProto.Column col1 = TableProto.Column.newBuilder().setString(CNY).build();
        TableProto.Column col2 = TableProto.Column.newBuilder().setInt64(0).build();
        TableProto.Column col3 = TableProto.Column.newBuilder().setInt64(0).build();
        TableProto.Column col4 = TableProto.Column.newBuilder().setString("system").build();//go中为BYTES
        TableProto.Column col5 = TableProto.Column.newBuilder().setInt64(System.currentTimeMillis()).build();
        List<TableProto.Column> cols = new ArrayList<TableProto.Column>();
        cols.add(col1);
        cols.add(col2);
        cols.add(col3);
        cols.add(col4);
        cols.add(col5);
        TableProto.Row rows = TableProto.Row.newBuilder()
                .addAllColumns(cols)
                .build();
        try {

            boolean success = false;

            success = stub.insertRow(TableCurrency, rows);

            if (success){
                log.info("Row CNY successfully inserted");
            }
        } catch (Exception e) {
            ret = e.toString();
            return ret;
        }
        //内置美元USD
        // 由于go使用了内存访问符 所以插入的时候特别方便，java只得用List转换
        col1 = TableProto.Column.newBuilder().setString(USD).build();
        col5 = TableProto.Column.newBuilder().setInt64(System.currentTimeMillis()).build();
        cols.clear();
        cols.add(col1);
        cols.add(col2);
        cols.add(col3);
        cols.add(col4);
        cols.add(col5);
        rows = TableProto.Row.newBuilder()
                .addAllColumns(cols)
                .build();
        try {

            boolean success = false;

            success = stub.insertRow(TableCurrency, rows);

            if (success){
                log.info("Row USD successfully inserted");
            }
        } catch (Exception e) {
            ret = e.toString();
            return ret;
        }


        return ret;
    }


    /*
    创建币
     */
    public String createCurrency(ChaincodeStub stub,String[] args){

        String ret = null;
        log.info("In  createCurrency...");
        if((args.length)!=3){
            ret = "Incorrect number of arguments. Expecting 3,in create currency";
            return ret;
        }
        String id = args[0];
        int count = Integer.parseInt(args[1]);
        String creator = getFromBASE64(args[2]);
        if(creator==null){
            ret = "Failed decodinf creator";
            return ret;
        }
        long timestamp = System.currentTimeMillis();
        TableProto.Column col1 = TableProto.Column.newBuilder().setString(id).build();
        TableProto.Column col2 = TableProto.Column.newBuilder().setInt64(count).build();
        TableProto.Column col3 = TableProto.Column.newBuilder().setInt64(count).build();
        TableProto.Column col4 = TableProto.Column.newBuilder().setString(creator).build();
        TableProto.Column col5 = TableProto.Column.newBuilder().setInt64(timestamp).build();
        List<TableProto.Column> cols = new ArrayList<TableProto.Column>();
        cols.add(col1);
        cols.add(col2);
        cols.add(col3);
        cols.add(col4);
        cols.add(col5);
        TableProto.Row row = TableProto.Row.newBuilder()
                .addAllColumns(cols)
                .build();
        try {

            boolean success = false;

            success = stub.insertRow(TableCurrency, row);

            if (success){
                log.info("create currency successfully inserted");
            }
        } catch (Exception e) {
            ret = e.toString();
            return ret;
        }
        if(count>0){
            cols.clear();
            cols.add(col1);
            cols.add(col2);
            cols.add(col5);
            row = TableProto.Row.newBuilder()
                    .addAllColumns(cols)
                    .build();
            try {

                boolean success = false;

                success = stub.insertRow(TableCurrencyReleaseLog, row);

                if (success){
                    log.info("createCurrency log successfully inserted");
                }
            } catch (Exception e) {
                ret = e.toString();
                return ret;
            }
        }
        log.info("Done  createCurrency...");


        return ret;
    }

    /*
     发布货币    未完待续。。。
     */
    public String releaseCurrency(ChaincodeStub stub,String[] args ){
        String ret = null;
        log.info("In  releaseCurrency...");
        if((args.length)!=2){
            ret = "Incorrect number of arguments. Expecting 3,in release currency";
            return ret;
        }
        String id = args[0];
        if(id.equals(CNY)||id.equals(USD)){
            ret = "Currency can't be CNY or USD";
            return ret;
        }
        TableProto.Row row = getTableCurrencRow(stub,id);
        if(row.equals(null)||row==null){
            return "getTableCurrencRow_ERROR";
        }
        String creator = row.getColumns(3).getString();
        if(creator.length()==0&&creator.equals(null)){
            ret = "Invalid creator,is null";
            return ret;
        }
        boolean verityCreator = isCreator(stub,row.getColumns(3).getString().getBytes());
        if(!verityCreator){
            ret = "Failed checking currency creator identity";
            return ret;
        }


        long count = Long.parseLong(args[1]);
        if(count<=0){
            ret = "The currency release count must be > 0";
            return ret;
        }

        long timestamp = System.currentTimeMillis();
        long newSumCount = row.getColumns(1).getInt64()+count;
        long newLeftCount = row.getColumns(2).getInt64()+count;

        List<TableProto.Column> cols = new ArrayList<TableProto.Column>();
        cols.add(TableProto.Column.newBuilder().setString(id).build());
        cols.add(TableProto.Column.newBuilder().setInt64(newSumCount).build());
        cols.add(TableProto.Column.newBuilder().setInt64(newLeftCount).build());
        cols.add(TableProto.Column.newBuilder().setString(creator).build());
        cols.add(TableProto.Column.newBuilder().setInt64(timestamp).build());

        TableProto.Row newRow = TableProto.Row.newBuilder()
                .addAllColumns(cols)
                .build();
        try {
            //go是直接更改内存信息replace列值，如果java也只replace list中的列，会不会有错？   待测试。。。

            boolean success = stub.replaceRow(TableCurrency, newRow);

            if (success){
                log.info("Failed replacing row");
            }
        } catch (Exception e) {
            ret = e.toString();
            return ret;
        }

        cols.clear();
        cols.add(TableProto.Column.newBuilder().setString(id).build());
        cols.add(TableProto.Column.newBuilder().setInt64(count).build());
        cols.add(TableProto.Column.newBuilder().setInt64(timestamp).build());

        TableProto.Row logRow = TableProto.Row.newBuilder()
                .addAllColumns(cols)
                .build();
        try {

            boolean success = stub.replaceRow(TableCurrencyReleaseLog, logRow);

            if (success){
                log.info("createCurrency log successfully inserted");
            }
        } catch (Exception e) {
            ret = e.toString();
            return ret;
        }

        log.info("Done releaseCurrency...");
        return ret;
    }


    /*
     分发币，
     */
    public String assignCurrency(ChaincodeStub stub,String[] args){
        String ret = null;
        log.info("In  Assign Currency...");
        if((args.length)!=1){
            ret = "Incorrect number of arguments. Expecting 3,in release currency";
        }
        /*
          json格式待确定，等待确定后继续。。。
         */
        String id = null;
        JSONObject jobj = JSONObject.fromObject(args[0]);
        if(jobj.has("currency")&&jobj.get("currency")!=null){
            id = jobj.get("currency").toString();
        }else{
            log.error("can't get assign's currency");
            return "can't get assign's currency";
        }

        JSONArray jarray;
        if(jobj.has("assigns")){
            jarray = jobj.getJSONArray("assigns");
        }else {
            return "Invalid assign data";
        }


        TableProto.Row row = getTableCurrencRow(stub,id);
        if(row==null||row.equals("")){
            ret = "Faild get row of id:"+id;
            return ret;
        }
        String creator = row.getColumns(3).getString();
        long leftCount = row.getColumns(2).getInt64();
        if(creator.length()==0||creator.equals(null)){
            ret = "Invalid creator,is null";
            return ret;
        }
        /*
        此函数由于源码未完善，带后续补充
         */
        boolean verityCreator = isCreator(stub,creator.getBytes());
        if(!verityCreator){
            ret = "Failed checking currency creator identity";
            return ret;
        }
        //

        long assignCount = 0;
        for(int j =0;j<jarray.size();j++){

            assignCount += Long.parseLong(jarray.getJSONObject(j).get("count").toString());
        }

        if(assignCount>leftCount){
            ret = "The left count:"+leftCount+" of currency:"+assignCount+
                    " is insufficient";
            return ret;
        }

        for(int k=jarray.size();k>0;k-- ){

            String owner = getFromBASE64(jarray.getJSONObject(k).get("owner").toString());
            long count = Long.parseLong(jarray.getJSONObject(k).get("count").toString());
            if(owner==null||owner.equals(" ")){
                ret = "Failed decodinfo owner";
                return ret;
            }
            if(count<=0)
                continue;
            List<TableProto.Column> cols = new ArrayList<TableProto.Column>();
            cols.add(TableProto.Column.newBuilder().setString(id).build());
            cols.add(TableProto.Column.newBuilder().setString(owner).build());
            cols.add(TableProto.Column.newBuilder().setInt64(count).build());
            cols.add(TableProto.Column.newBuilder().setInt64(System.currentTimeMillis()).build());

            TableProto.Row assignRow = TableProto.Row.newBuilder()
                    .addAllColumns(cols)
                    .build();
            try {

                boolean success = stub.insertRow(TableCurrencyAssignLog, assignRow);

                if (success){
                    log.info("create TableCurrencyAssign log successfully inserted");
                }
            } catch (Exception e) {
                ret = e.toString();
                return ret;
            }

            TableProto.Row assetRow = getOwnerOneAsset(stub,owner,id);
            if(assetRow==null||assetRow.equals("")){
                ret = "Faild get row of id:"+id;
                return ret;
            }

            cols.clear();
            cols.add(TableProto.Column.newBuilder().setString(owner).build());
            cols.add(TableProto.Column.newBuilder().setString(id).build());
            cols.add(TableProto.Column.newBuilder().setInt64(count).build());
            cols.add(TableProto.Column.newBuilder().setInt64(0).build());

            TableProto.Row atRow = TableProto.Row.newBuilder()
                    .addAllColumns(cols)
                    .build();
            if(assetRow.getColumnsCount()==0){

                try {

                    stub.insertRow(TableAssets, atRow);

                } catch (Exception e) {
                    ret = e.toString();
                    return ret;
                }

            }else{
                try {
                    long asset_count = assetRow.getColumns(2).getInt64();
                    long asset_lockcount = assetRow.getColumns(4).getInt64();
                    cols.clear();
                    cols.add(TableProto.Column.newBuilder().setString(owner).build());
                    cols.add(TableProto.Column.newBuilder().setString(id).build());
                    cols.add(TableProto.Column.newBuilder().setInt64(count+asset_count).build());
                    cols.add(TableProto.Column.newBuilder().setInt64(asset_lockcount).build());

                    stub.replaceRow(TableAssets, atRow);

                } catch (Exception e) {
                    ret = e.toString();
                    return ret;
                }

            }

            cols.clear();
            leftCount -= count;
        }

        /*
        每次replaceRow的时候可以看到
        java需要把整个list给替换了  而go只修改row中更改的内存信息
         */
        if(leftCount!=row.getColumns(2).getInt64()){
            List<TableProto.Column> cs = new ArrayList<TableProto.Column>();
            cs.add(TableProto.Column.newBuilder().setString(id).build());
            cs.add(row.getColumns(1));
            cs.add(TableProto.Column.newBuilder().setInt64(leftCount).build());
            cs.add(TableProto.Column.newBuilder().setString(creator).build());
            cs.add(row.getColumns(4));

            TableProto.Row tcRow = TableProto.Row.newBuilder()
                    .addAllColumns(cs)
                    .build();
            try {

                stub.replaceRow(TableCurrency, tcRow);

            } catch (Exception e) {
                ret = e.toString();
                return ret;
            }

        }

        log.info("Done  Assign Currency...");
        return  ret;
    }

    /*
    **********
    * ********由于 java-chaincodeEvent还不能使用，跟其有关的场合暂不做业务处理
     */
    public  String lock(ChaincodeStub stub,String[] args){

        String ret = null;
        log.info("In  lock Currency...");
        if(args.length!=3){
            return "Incorrect number of arguments. Excepcting 3";
        }
        //String owner,currency,orderId;
        // long count;

        JSONArray jsonArray = JSONArray.fromObject(args[0]);
        boolean islock = Boolean.getBoolean(args[1]);
        List successIn = new ArrayList<String>();
        List failIn    = new ArrayList<String>();
        for(int j =0;j<jsonArray.size();j++){
            String owner = getFromBASE64(jsonArray.getJSONObject(j).getString("owner"));
            String currency = jsonArray.getJSONObject(j).getString("currency");
            String orderId = jsonArray.getJSONObject(j).getString("orderId");
            long count = Long.parseLong(jsonArray.getJSONObject(j).getString("count"));

            if(owner==null){
                log.error("lock error2");
                failIn.add(orderId);
                continue;
            }
            String ret_loub = lockOrUnlockBalance(stub,owner,currency,orderId,count,islock);
            if(ret_loub!=null){
                if(ret_loub.equals("-1")){
                    failIn.add(orderId);
                    continue;
                }else{
                    log.error("lock error3");
                    return null;
                }

            }
            successIn.add(orderId);

        }
        /*******
         *
         *   此处留给javaChaincodeEvent所用！！
         */

        return ret;
    }

    /*
    ***********
    * *********   这里因为chaincodeEvent未完善 所以待续
    * *********
     */
    public String exchange(ChaincodeStub stub,String[] args){
        String ret = null;
        log.info("In  Exchange...");
        if(args.length!=1){
            return "Incorrect number of arguments. Exception 2";
        }

        //JSONObject json = JSONObject.fromObject(args[0].toString());

        /*
        go 里面解析json非常方便！
         */
//        String UUID = json.get("uuid").toString();                      //UUID
//        String Account = json.get("account").toString();;               //账户
//        String SrcCurrency  = json.get("srcCurrency").toString();       //源币种代码
//        String DesCurrency  = json.get("desCurrency").toString();       //目标币种代码
//        String RawUUID  = json.get("rawUUID").toString();               //母单UUID
//        String Metadata  = json.get("metadata").toString();             //存放其他数据，如挂单锁定失败信息
//        long SrcCount  = Long.parseLong(json.get("srcCount").toString());           //源币种交易数量
//        long DescCount= Long.parseLong(json.get("desCount").toString());         //目标币种交易数量
//        long ExpiredTime= Long.parseLong(json.get("expiredTime").toString());        //超时时间
//        long PendingTime= Long.parseLong(json.get("PendingTime").toString());        //提交挂单时间
//        long PendedTime= Long.parseLong(json.get("PendedTime").toString());         //挂单完成时间
//        long MatchedTime= Long.parseLong(json.get("matchedTime").toString());        //撮合完成时间
//        long FinishedTime= Long.parseLong(json.get("finishedTime").toString());       //交易完成时间
//        boolean IsbuyAll= (boolean) json.get("isBuyAll");          //是否买入所有，即为true是以目标币全部兑完为主,
//                                                                    // 否则算部分成交,买完为止；为false则是以源币全部兑完为主,
//                                                                    // 否则算部分成交，卖完为止


        /*
        GO中转换嵌套json非常容易，而java中的json库则相对来说没那么容易
         */
        JSONArray jsonArray = JSONArray.fromObject(args[0]);
        if(jsonArray.size()==0||jsonArray==null){
            log.error("exchange error1");
            return "args invalid..";
        }
        List successIn = new ArrayList<String>();
        List failIn = new ArrayList<String>();
        for(int i = 0;i<jsonArray.size();i++){
            JSONObject buyOrder = JSONObject.fromObject(jsonArray.getJSONObject(i).get("buyOrder"));
            JSONObject sellOrder = JSONObject.fromObject(jsonArray.getJSONObject(i).get("sellOrder"));


            //调用其他方法避免多次base64解码
            String buyOwner = getFromBASE64(buyOrder.getString("account"));
            String sellOwner = getFromBASE64(sellOrder.getString("account"));
            buyOrder.put("account",buyOwner);
            sellOrder.put("account",sellOwner);

            String matchOrder = buyOrder.getString("UUID")+","+sellOrder.getString("UUID");

            if(buyOrder.getString("SrcCurrency")!=sellOrder.getString("DesCurrency")
                    ||buyOrder.getString("DesCurrency")!=sellOrder.getString("SrcCurrency")){
                return "The exchange is invalid";
            }

            TableProto.Row buyRow = getTxLogByID(stub,buyOrder.getString("UUID"));
            TableProto.Row sellRow = getTxLogByID(stub,sellOrder.getString("UUID"));

            if(buyRow!=null&&buyRow.getColumnsCount()>0||sellRow!=null&&sellRow.getColumnsCount()>0){
                log.error("exchange error2");
                failIn.add(matchOrder);

                continue;
            }
            String retETx = execTx(stub,buyOrder,sellOrder);
            if(retETx!=null){
                log.error("exchange error3");
                if(retETx.equals("-1")){
                    failIn.add(matchOrder);

                    continue;
                }else{
                    log.error("exchange error4");
                    return  ret;
                }
            }
            String retSTL = saveTxLog(stub,buyOrder,sellOrder);
            if(retSTL!=null){
                log.error("exchange error5");
                return ret;
            }
            successIn.add(matchOrder);

        }

        /**********  挂单成功或失败
         *
         * 此处留用写事件！！！ chaincodeEvent
         */



        return ret;
    }
    /*
     -1 对应go的 CheckErr
     -2 对应go的 WorldStateErr
     */
    public String execTx(ChaincodeStub stub,JSONObject jbuy,JSONObject jsell){
        // 买完为止的挂单结算结余数量
        // 挂单UUID等于原始ID时表示该单交易完成
        String ret = null;
        String bAccount = jbuy.getString("Account");
        String bSrcCurrency = jbuy.getString("SrcCurrency");
        String bDesCurrency = jbuy.getString("DesCurrency");
        String bRawUUID = jbuy.getString("RawUUID");
        long bDesCount =jbuy.getLong("DesCount");
        long bFinalCost = jbuy.getLong("FinalCost");
        String sAccount = jsell.getString("Account");
        String sSrcCurrency = jsell.getString("SrcCurrency");
        String sDesCurrency = jsell.getString("DesCurrency");
        String sRawUUID = jsell.getString("RawUUID");
        long sDesCount = jsell.getLong("DesCount");
        long sFinalCost = jsell.getLong("FinalCost");
        if(jbuy.getBoolean("IsBuyAll") && jbuy.get("UUID")==bRawUUID) {

            long lockCount = computeBalance(stub, bAccount, bSrcCurrency,
                    bDesCurrency, bRawUUID, bFinalCost);
            if(lockCount==-1){
                return "-1";
            }

            log.debug("Order " + jbuy.getString("UUID") + " balance " + lockCount);
            if (lockCount > 0) {
                String check = lockOrUnlockBalance(stub, bAccount, bSrcCurrency,
                        bRawUUID, lockCount, false);
                if (check != null) {
                    log.error("execTx error2");
                    return "Failed unlock balance";
                }

            }
        }
        // 买单源币锁定数量减少
        TableProto.Row buySrcRow = getOwnerOneAsset(stub,bAccount,bSrcCurrency);
        if(buySrcRow==null||buySrcRow.getColumnsCount()==0){
            log.error("execTx error3");
            return "-1 ";
        }

        List<TableProto.Column> cols = new ArrayList<TableProto.Column>();
        cols.add(buySrcRow.getColumns(0));
        cols.add(buySrcRow.getColumns(1));
        cols.add(buySrcRow.getColumns(2));
        cols.add(TableProto.Column.newBuilder().setInt64
                (buySrcRow.getColumns(2).getInt64()-bFinalCost).build());
        TableProto.Row newBuySrcRow = TableProto.Row.newBuilder()
                .addAllColumns(cols)
                .build();
        try {

            boolean success = false;

            success = stub.replaceRow(TableAssets, newBuySrcRow);

            if (success){
                log.info("Row TableAssets successfully replaced");
            }
        } catch (Exception e) {
            ret = e.toString();
            log.error("execTx error4"+ret);
            return "-2";
        }
        // 买单目标币数量增加
        TableProto.Row buyDesRow = getOwnerOneAsset(stub,bAccount,bDesCurrency);
        if(buyDesRow==null){
            log.error("execTx error5");
            return "-1 ";
        }

        cols.clear();
        if(buyDesRow.getColumnsCount()==0){
            cols.add(TableProto.Column.newBuilder()
                    .setString(bAccount)
                    .build());
            cols.add(TableProto.Column.newBuilder()
                    .setString(bDesCurrency)
                    .build());
            cols.add(TableProto.Column.newBuilder()
                    .setInt64(bDesCount)
                    .build());
            cols.add(TableProto.Column.newBuilder()
                    .setInt64(0)
                    .build());
            TableProto.Row newBDrow = TableProto.Row.newBuilder()
                    .addAllColumns(cols)
                    .build();
            try {

                boolean success = false;

                success = stub.insertRow(TableAssets, newBDrow);

                if (success){
                    log.info("newBDrow  successfully inserted");
                }
            } catch (Exception e) {
                ret = e.toString();
                log.error("execTx error6 "+ret);
                return "-2";
            }

        }else{
            cols.add(buyDesRow.getColumns(0));
            cols.add(buyDesRow.getColumns(1));
            cols.add(TableProto.Column.newBuilder()
                    .setInt64(buyDesRow.getColumns(2).getInt64()+bDesCount)
                    .build());
            cols.add(buyDesRow.getColumns(3));
            TableProto.Row newBDrow = TableProto.Row.newBuilder()
                    .addAllColumns(cols)
                    .build();
            try {

                boolean success = false;

                success = stub.replaceRow(TableAssets, newBDrow);

                if (success){
                    log.info("newBDrow  successfully replaced");
                }
            } catch (Exception e) {
                ret = e.toString();
                log.error("execTx error7 "+ret);
                return "-2";
            }


        }
        // 买完为止的挂单结算结余数量
        // 挂单UUID等于原始ID时表示该单交易完成
        if(jsell.getBoolean("IsBuyAll") && jsell.get("UUID")==sRawUUID){

            long unlockCount = computeBalance(stub,sAccount,sSrcCurrency,
                    sDesCurrency,sRawUUID,sFinalCost);
            if(unlockCount>0){
                String check = lockOrUnlockBalance(stub,sAccount,sSrcCurrency,
                        sRawUUID,unlockCount,false);
                log.debug("Order "+jsell.getString("UUID")+" balance "+unlockCount);
                if(check!=null){
                    log.error("execTx error9");
                    return "-1";
                }

            }

        }

        // 卖单源币数量减少
        TableProto.Row sellSrcRow = getOwnerOneAsset(stub,sAccount,sSrcCurrency);
        if(sellSrcRow==null||sellSrcRow.getColumnsCount()==0){
            log.error("execTx error10");
            return "the user have not currency "+sSrcCurrency;
        }
        cols.clear();
        cols.add(sellSrcRow.getColumns(0));
        cols.add(sellSrcRow.getColumns(1));
        cols.add(sellSrcRow.getColumns(2));
        cols.add(TableProto.Column.newBuilder().setInt64
                (sellSrcRow.getColumns(2).getInt64()-sFinalCost).build());
        TableProto.Row newSellSrcRow = TableProto.Row.newBuilder()
                .addAllColumns(cols)
                .build();
        try {

            boolean success = false;
            success = stub.replaceRow(TableAssets, newSellSrcRow);

            if (success){
                log.info("Row TableAssets successfully replaced");
            }
        } catch (Exception e) {
            ret = e.toString();
            log.error("execTx error11"+ret);
            return ret;
        }

        // 卖单目标币数量增加
        TableProto.Row sellDesRow = getOwnerOneAsset(stub,sAccount,sDesCurrency);
        if(sellDesRow==null){
            log.error("execTx error12");
            return "Faild retrieving asset "+sDesCurrency;
        }
        cols.clear();
        if(sellDesRow.getColumnsCount()==0){
            cols.add(TableProto.Column.newBuilder()
                    .setString(sAccount)
                    .build());
            cols.add(TableProto.Column.newBuilder()
                    .setString(sDesCurrency)
                    .build());
            cols.add(TableProto.Column.newBuilder()
                    .setInt64(sDesCount)
                    .build());
            cols.add(TableProto.Column.newBuilder()
                    .setInt64(0)
                    .build());
            TableProto.Row newSDrow = TableProto.Row.newBuilder()
                    .addAllColumns(cols)
                    .build();
            try {

                boolean success = stub.insertRow(TableAssets, newSDrow);

                if (success){
                    log.info("newSDrow  successfully inserted");
                }
            } catch (Exception e) {
                ret = e.toString();
                log.error("execTx error13 "+ret);
                return ret;
            }

        }else{
            cols.add(sellDesRow.getColumns(0));
            cols.add(sellDesRow.getColumns(1));
            cols.add(TableProto.Column.newBuilder()
                    .setInt64(sellDesRow.getColumns(2).getInt64()+sDesCount)
                    .build());
            cols.add(sellDesRow.getColumns(3));
            TableProto.Row newSDrow = TableProto.Row.newBuilder()
                    .addAllColumns(cols)
                    .build();
            try {

                boolean success = stub.replaceRow(TableAssets, newSDrow);

                if (success){
                    log.info("newBDrow  successfully replaced");
                }
            } catch (Exception e) {
                ret = e.toString();
                log.error("execTx error7 "+ret);
                return ret;
            }

        }

        return ret;
    }

    public String saveTxLog(ChaincodeStub stub,JSONObject jbuy,JSONObject jsell){
        String ret = null;

        List<TableProto.Column> cols = new ArrayList<TableProto.Column>();
        cols.add(TableProto.Column.newBuilder()
                .setString(jbuy.getString("Account"))
                .build());
        cols.add(TableProto.Column.newBuilder()
                .setString(jbuy.getString("SrcCurrency"))
                .build());
        cols.add(TableProto.Column.newBuilder()
                .setString(jbuy.getString("DesCurrency"))
                .build());
        cols.add(TableProto.Column.newBuilder()
                .setString(jbuy.getString("RawUUID"))
                .build());
        cols.add(TableProto.Column.newBuilder()
                .setString(jbuy.toString())
                .build());
        TableProto.Row row = TableProto.Row.newBuilder()
                .addAllColumns(cols)
                .build();
        try {

            boolean success = stub.insertRow(TableTxLog, row);

            if (success){
                log.info("TableTxLog ROW  successfully inserted");
            }
        } catch (Exception e) {
            ret = e.toString();
            log.error("saveTxLog error1 "+ret);
            return ret;
        }
        cols.clear();
        cols.add(TableProto.Column.newBuilder()
                .setString(jbuy.getString("UUID"))
                .build());
        cols.add(TableProto.Column.newBuilder()
                .setString(jbuy.toString())
                .build());
        row = TableProto.Row.newBuilder()
                .addAllColumns(cols)
                .build();
        try {

            boolean success = stub.insertRow(TableTxLog2, row);

            if (success){
                log.info("TableTxLog2 ROW  successfully inserted");
            }
        } catch (Exception e) {
            ret = e.toString();
            log.error("saveTxLog error2 "+ret);
            return ret;
        }
        cols.clear();
        cols.add(TableProto.Column.newBuilder()
                .setString(jsell.getString("Account"))
                .build());
        cols.add(TableProto.Column.newBuilder()
                .setString(jsell.getString("SrcCurrency"))
                .build());
        cols.add(TableProto.Column.newBuilder()
                .setString(jsell.getString("DesCurrency"))
                .build());
        cols.add(TableProto.Column.newBuilder()
                .setString(jsell.getString("RawUUID"))
                .build());
        cols.add(TableProto.Column.newBuilder()
                .setString(jsell.toString())
                .build());
        row = TableProto.Row.newBuilder()
                .addAllColumns(cols)
                .build();
        try {

            boolean success = stub.insertRow(TableTxLog, row);

            if (success){
                log.info("TableTxLog ROW  successfully inserted");
            }
        } catch (Exception e) {
            ret = e.toString();
            log.error("saveTxLog error3 "+ret);
            return ret;
        }

        cols.clear();
        cols.add(TableProto.Column.newBuilder()
                .setString(jsell.getString("UUID"))
                .build());
        cols.add(TableProto.Column.newBuilder()
                .setString(jsell.toString())
                .build());
        row = TableProto.Row.newBuilder()
                .addAllColumns(cols)
                .build();
        try {

            boolean success = stub.insertRow(TableTxLog2, row);

            if (success){
                log.info("TableTxLog2 ROW  successfully inserted");
            }
        } catch (Exception e) {
            ret = e.toString();
            log.error("saveTxLog error4 "+ret);
            return ret;
        }

        return ret;
    }

    //计算挂单结余
    public long computeBalance(ChaincodeStub stub,String owner,String srcCurrency,
                               String desCurrency,String rawUUID,long currentCost){
        long ret=0;
        TableProto.Row logRow = getLockLog(stub,owner,srcCurrency,rawUUID,true);
        if(logRow==null||logRow.getColumnsCount()==0){
            log.error("get locklog faild");
            return -1;
        }
        long sumcost = 0;
        synchronized (this){
            ArrayList<TableProto.Row> txRows = getTXs(stub,owner,srcCurrency,desCurrency,rawUUID);
            for(TableProto.Row row:txRows){
                JSONObject jobj = JSONObject.fromObject(row.getColumns(4).getString());
                sumcost += Long.parseLong(jobj.get("FinalCost").toString());
            }

        }

        long lockCount = logRow.getColumns(4).getInt64();
        return lockCount-sumcost-currentCost;
    }

    public ArrayList<TableProto.Row> getTXs(ChaincodeStub stub,String owner,
                                            String srcCurrency,String desCurrency,String rawOrder){

        TableProto.Column queryCol =
                TableProto.Column.newBuilder()
                        .setString(owner).build();
        TableProto.Column queryCol1=
                TableProto.Column.newBuilder()
                        .setString(srcCurrency).build();
        TableProto.Column queryCol2 =
                TableProto.Column.newBuilder()
                        .setString(desCurrency).build();
        TableProto.Column queryCol3 =
                TableProto.Column.newBuilder()
                        .setString(rawOrder).build();
        List<TableProto.Column> key = new ArrayList<TableProto.Column>();
        key.add(queryCol);
        key.add(queryCol1);
        key.add(queryCol2);
        key.add(queryCol3);
        ArrayList<TableProto.Row> rows = null;
        try {
            rows = stub.getRows(TableTxLog,key);
        }catch(Exception e){
            return null;
        }

        return rows;

    }

    public TableProto.Row getTxLogByID(ChaincodeStub stub,String uuid){

        TableProto.Column queryCol =
                TableProto.Column.newBuilder()
                        .setString(uuid).build();
        List<TableProto.Column> key = new ArrayList<TableProto.Column>();
        key.add(queryCol);
        TableProto.Row row = null;
        try {
            row = stub.getRow(TableTxLog2, key);
        }catch(Exception e){
            return null;
        }

        return row;

    }

    //622 line

    /*
     锁定货币
     */
    public  String lockBalance(ChaincodeStub stub,String[] args){

        String ret = null;
        log.info("In  lock Currency...");
        if(args.length!=3){
            return "Incorrect number of arguments. Excepcting 3";
        }
        String owner = getFromBASE64(args[0].toString());
        if(owner==null||owner.equals("")){
            return "Failed decode owner";
        }
        String id = args[1].toString();
        long count = Long.parseLong(args[2].toString());
        String order = args[3].toString();
        if(count<=0){
            return "count:"+count+" is invaild";
        }
        log.info("Done  lock Currency...");

        return ret;
    }

    /*
     解锁货币
     */
    public  String unlockBalance(ChaincodeStub stub,String[] args){

        String ret = null;
        log.info("In  unlock Currency...");
        if(args.length!=3){
            return "Incorrect number of arguments. Excepcting 3";
        }
        String owner = getFromBASE64(args[0].toString());
        if(owner==null||owner.equals("")){
            return "Failed decode owner";
        }
        String id = args[1].toString();
        long count = Long.parseLong(args[2].toString())*(-1);
        String order = args[3].toString();

        if(count>=0){
            return "count:"+count+" is invaild";
        }
        log.info("Done  unlock Currency...");

        return ret;
    }

    public  String lockOrUnlockBalance(ChaincodeStub stub, String owner,String currency,
                                       String order,long count,boolean islock){

        String ret = null;
        TableProto.Row row = getOwnerOneAsset(stub,owner,currency);
        if(row==null||row.equals("")||row.getColumnsCount()==0){
            ret = "Faild get row of id:"+currency;
            return "-1";
        }

        long currencyCount = row.getColumns(2).getInt64();
        long currencyLockCount = row.getColumns(3).getInt64();
        if(islock && currencyCount<count||!islock && currencyLockCount<count){
            return "-1";
        }

        // 判断是否锁定过，应为是批量操作，可能会有重复数据。其他批量操作也要作此判断
        TableProto.Row lockRow = getLockLog(stub,owner,currency,order,islock);
        if(lockRow !=null && lockRow.getColumnsCount()>0){
            return " -1";
        }

        List<TableProto.Column> cs = new ArrayList<TableProto.Column>();
        cs.add(TableProto.Column.newBuilder().setString(owner).build());
        cs.add(TableProto.Column.newBuilder().setString(currency).build());

        if ( islock==true ){

            cs.add(TableProto.Column.newBuilder().setInt64(currencyCount-count).build());
            cs.add(TableProto.Column.newBuilder().setInt64(currencyLockCount+count).build());
        }else{

            cs.add(TableProto.Column.newBuilder().setInt64(currencyCount+count).build());
            cs.add(TableProto.Column.newBuilder().setInt64(currencyLockCount-count).build());
        }


        TableProto.Row taRow = TableProto.Row.newBuilder()
                .addAllColumns(cs)
                .build();
        try {

            stub.replaceRow(TableAssets, taRow);

        } catch (Exception e) {
            ret = e.toString();
            return "-2";
        }

        cs.clear();
        cs.add(TableProto.Column.newBuilder().setString(owner).build());
        cs.add(TableProto.Column.newBuilder().setString(currency).build());
        cs.add(TableProto.Column.newBuilder().setString(order).build());
        cs.add(TableProto.Column.newBuilder().setBool(islock).build()); //此前我这里逻辑为当count>0为true，somehow？ go版本的变动
        cs.add(TableProto.Column.newBuilder().setInt64(count).build());
        cs.add(TableProto.Column.newBuilder().setInt64(System.currentTimeMillis()).build());

        TableProto.Row tallRow = TableProto.Row.newBuilder()
                .addAllColumns(cs)
                .build();
        try {

            stub.replaceRow(TableAssetLockLog, tallRow);

        } catch (Exception e) {
            ret = e.toString();
            return "-2";
        }

        return ret;
    }




    /*
     // In order to enforce access control, we require that the
	// metadata contains the following items:
	// 1. a certificate Cert
	// 2. a signature Sigma under the signing key corresponding
	// to the verification key inside Cert of :
	// (a) Cert;
	// (b) The payload of the transaction (namely, function name and args) and
	// (c) the transaction binding.

	// Verify Sigma=Sign(certificate.sk, Cert||tx.Payload||tx.Binding) against Cert.vk
     */
    public boolean isCreator(ChaincodeStub stub, byte[] certificate){


        byte[] sigma = stub.getCallerMetadata();
        byte[] payload = stub.getPayload();
        // byte[] bingding = stub.getBinding();

        if(sigma.length==0||certificate.length==0||payload.length==0){
            log.error("get securitycontext failed");
            return false;

        }
        boolean verity = stub.verifySignature(certificate,sigma,payload);

        if(!verity){
            log.error("invalid signature");
        }

        return verity;
    }


    //获取table中的row
    public TableProto.Row getTableCurrencRow(ChaincodeStub stub,String id){
        TableProto.Column queryCol =
                TableProto.Column.newBuilder()
                        .setString(id).build();
        List<TableProto.Column> key = new ArrayList<TableProto.Column>();
        key.add(queryCol);
        TableProto.Row row = null;
        try {
            row = stub.getRow(TableCurrency, key);
        }catch(Exception e){
            return null;
        }

        return row;
    }

    public TableProto.Row getOwnerOneAsset(ChaincodeStub stub,String owner,String currency){
        TableProto.Column queryCol =
                TableProto.Column.newBuilder()
                        .setString(owner).build();
        TableProto.Column queryCol1 =
                TableProto.Column.newBuilder()
                        .setString(currency).build();
        List<TableProto.Column> key = new ArrayList<TableProto.Column>();
        key.add(queryCol);
        key.add(queryCol1);
        TableProto.Row row = null;
        try {
            row = stub.getRow(TableAssets,key);

        }catch(Exception e){
            return null;

        }

        return row;
    }

    public TableProto.Row getLockLog(ChaincodeStub stub,String owner,String currency,String order,boolean islock){
        TableProto.Column queryCol =
                TableProto.Column.newBuilder()
                        .setString(owner).build();
        TableProto.Column queryCol1 =
                TableProto.Column.newBuilder()
                        .setString(currency).build();
        TableProto.Column queryCol2 =
                TableProto.Column.newBuilder()
                        .setString(order).build();
        TableProto.Column queryCol3 =
                TableProto.Column.newBuilder()
                        .setBool(islock).build();
        List<TableProto.Column> key = new ArrayList<TableProto.Column>();
        key.add(queryCol);
        key.add(queryCol1);
        key.add(queryCol2);
        key.add(queryCol3);
        TableProto.Row row = null;
        try {
            row = stub.getRow(TableAssetLockLog,key);

        }catch(Exception e){
            return null;

        }

        return row;
    }





    public  String getFromBASE64(String s) {
        if (s == null) return null;
        BASE64Decoder decoder = new BASE64Decoder();
        try {
            byte[] b = decoder.decodeBuffer(s);
            return new String(b);
        } catch (Exception e) {
            return null;
        }
    }

    class Order{
        String UUID;             //UUID
        String Account;          //账户
        String SrcCurrency;      //源币种代码
        String DesCurrency;      //目标币种代码
        String RawUUID;          //母单UUID
        String Metadata;         //存放其他数据，如挂单锁定失败信息
        long SrcCount;           //源币种交易数量
        long DescCount;          //目标币种交易数量
        long ExpiredTime;        //超时时间
        long PendingTime;        //提交挂单时间
        long PendedTime;         //挂单完成时间
        long MatchedTime;        //撮合完成时间
        long FinishedTime;       //交易完成时间
        boolean IsbuyAll;        //是否买入所有，即为true是以目标币全部兑完为主,
        // 否则算部分成交,买完为止；为false则是以源币全部兑完为主,
        // 否则算部分成交，卖完为止


    }

    @Override
    public String query(ChaincodeStub stub, String function, String[] args) {

        String ret = null;
        if(function.equals("createCurrency")){
            return createCurrency(stub,args);
        }else if(function.equals("releaseCurrency")){
            return releaseCurrency(stub,args);
        }else if(function.equals("assignCurrency")){
            return assignCurrency(stub,args);
        }else if(function.equals("exchange")){
            return exchange(stub,args);
        }else if(function.equals("lock")){
            return lock(stub,args);
        }else if(function.equals("queryCurrencyByID")){
            return queryCurrencyByID(stub,args);
        }else if(function.equals("queryAllCurrency")){
            return queryAllCurrency(stub,args);
        }else if(function.equals("queryTxLogs")){
            return queryTxLogs(stub,args);
        }else if(function.equals("queryAssetByOwner")){
            return queryAssetByOwner(stub,args);

        }

        return ret;
    }
    public String queryCurrencyByID(ChaincodeStub stub,String[] args){

        log.debug("queryCurrencyByID...");
        if(args.length!=1){
            return "Incorrect number of arguments. Expecting 1";
        }
        String id = args[0];
        TableProto.Row cuRow = getCurrencyByID(stub,id);
        if(cuRow==null){
            log.error("queryCurrencyByID error1");
            return "queryCurrencyByID error1";
        }
        if(cuRow.getColumnsCount()==0){
            return "no data can be queryed";
        }
        JSONObject cuJson = new JSONObject();
        cuJson.put("id",cuRow.getColumns(0).getString());
        cuJson.put("count",cuRow.getColumns(1).getInt64());
        cuJson.put("leftCount",cuRow.getColumns(2).getInt64());
        cuJson.put("creator",cuRow.getColumns(3).getString());
        cuJson.put("createTime",cuRow.getColumns(4).getInt64());

        return cuJson.toString();
    }
    TableProto.Row getCurrencyByID(ChaincodeStub stub,String id){

        TableProto.Column queryCol =
                TableProto.Column.newBuilder()
                        .setString(id).build();
        List<TableProto.Column> key = new ArrayList<TableProto.Column>();
        key.add(queryCol);
        TableProto.Row row = null;
        try {
            row = stub.getRow(TableAssetLockLog,key);

        }catch(Exception e){
            return null;

        }
        return row;


    }
    public String queryAllCurrency(ChaincodeStub stub,String[] args){

        log.debug("queryAllCurrency...");
        if(args.length!=0){
            log.error("incorrect number of arguments");
            return "incorrect number of arguments";
        }
        ArrayList<TableProto.Row> rows = null;
        try {
            rows = stub.getRows(TableCurrency,null);
        }catch(Exception e){
            return "getRows operation failed";
        }
        JSONArray jArray = new JSONArray();
        for(int i= 0;i<rows.size();i++){
            JSONObject jObject = new JSONObject();
            jObject.put("id",rows.get(i).getColumns(0).getString());
            jObject.put("count",rows.get(i).getColumns(1).getInt64());
            jObject.put("leftCount",rows.get(i).getColumns(2).getInt64());
            jObject.put("creator",rows.get(i).getColumns(3).getString());
            jObject.put("createTime",rows.get(i).getColumns(4).getInt64());
            jArray.add(jObject);
        }
        if(jArray.size()==0){
            return "no data can be queryed";
        }

        return jArray.toString();
    }
    public String queryTxLogs(ChaincodeStub stub,String[] args){

        log.debug("queryTxLogs...");
        if(args.length!=0){
            return "Incorrect number of arguments";
        }
        ArrayList<TableProto.Row> rows = null;
        try {
            rows = stub.getRows(TableTxLog2,null);
        }catch(Exception e){
            return "getRows operation failed";
        }
        JSONArray jArray = new JSONArray();
        for(int i=0;i<rows.size();i++){
            JSONObject jObj = JSONObject.fromObject(rows.get(i).getColumns(1));
            jArray.add(jObj);
        }
        if(jArray.size()==0){
            return "no data can be queryed of TableTxLog2";
        }
        return null;
    }

    // queryAssetByOwner 查询个人资产
    public String queryAssetByOwner(ChaincodeStub stub,String[] args){

        log.debug("queryAssetByOwner...");
        if(args.length!=1){
            return "Incorrect number of aragument. Expecting 1";
        }
        String owner = getFromBASE64(args[0]);
        if(owner==null){
            log.error("queryAssetByOwner error1");
            return "Failed decode owner";
        }
        ArrayList<TableProto.Row> rows = getOwnerAllAsset(stub,owner);
        if(rows==null){
            log.error("queryAssetByOwner error2");
            return null;
        }
        JSONArray jArray = new JSONArray();
        for(int i=0;i<rows.size();i++){
            JSONObject jObj = new JSONObject();
            jObj.put("owner",rows.get(i).getColumns(0).getString());
            jObj.put("currency",rows.get(i).getColumns(1).getString());
            jObj.put("count",rows.get(i).getColumns(2).getInt64());
            jObj.put("lockCount",rows.get(i).getColumns(3).getInt64());
            jArray.add(jObj);

        }

        return jArray.toString();
    }
    public ArrayList<TableProto.Row> getOwnerAllAsset(ChaincodeStub stub,String owner){

        TableProto.Column queryCol =
                TableProto.Column.newBuilder()
                        .setString(owner).build();
        List<TableProto.Column> key = new ArrayList<TableProto.Column>();
        key.add(queryCol);
        ArrayList<TableProto.Row> rows = null;
        try {
            rows = stub.getRows(TableAssets,key);
        }catch(Exception e){
            return null;
        }
        return rows;

    }


    @Override
    public String getChaincodeID() {
        return null;
    }

    public static void main(String[] args) throws Exception {

        new CryptoPrimitives("SHA3",256);
        System.out.println("Hello world! starting "+args);
        log.info("starting");
        new KRCC().start(args);
    }
}
